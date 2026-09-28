# Windows 下的开发任务脚本（替代 Makefile，因为本机没装 make）
# 用法：.\dev.ps1 [run|build|test|cover|fmt|tidy|lint|db-init|migrate-up|migrate-down|migrate-version|migrate-force]
#
# 迁移相关：
#   .\dev.ps1 db-init                    # 【首次】用超级用户装扩展（一次性）
#   .\dev.ps1 migrate-up                 # 应用所有未执行的迁移
#   .\dev.ps1 migrate-down               # 回滚 1 步
#   .\dev.ps1 migrate-down -Steps 3      # 回滚 3 步
#   .\dev.ps1 migrate-version            # 查看当前版本
#   .\dev.ps1 migrate-force -DbVersion 6 # 强制设定版本（不执行 SQL）
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        'run', 'build', 'test', 'cover', 'fmt', 'tidy', 'lint',
        'db-init', 'migrate-up', 'migrate-down', 'migrate-version', 'migrate-force'
    )]
    [string]$Task = 'run',

    # migrate-down 的回滚步数
    [int]$Steps = 1,

    # migrate-force 的目标版本
    # 【注意】不能叫 $Version —— 下面还有 `$version = 'dev'` 用于构建信息，
    # PowerShell 变量名不区分大小写，会撞成同一个变量并触发类型转换错误。
    [int]$DbVersion = -1
)

$ErrorActionPreference = 'Stop'

# ---------- 环境自检 ----------
$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host '[错误] 找不到 go 命令。' -ForegroundColor Red
    Write-Host '       Go 装在 D:\dev\go。请先重启终端让环境变量生效，'
    Write-Host '       或在当前窗口临时执行： $env:Path = "D:\dev\go\bin;" + $env:Path'
    exit 1
}

# 国内网络必须走代理，否则拉依赖会连 proxy.golang.org 超时
if (-not $env:GOPROXY) { $env:GOPROXY = 'https://goproxy.cn,direct' }

$Module = 'github.com/mzzzmnq/leetnote-api'

# ---------- 版本信息（git 不可用时回退默认值）----------
$version = 'dev'
$commit  = 'none'
try { $version = (git describe --tags --always --dirty 2>$null) } catch { }
try { $commit  = (git rev-parse --short HEAD 2>$null) } catch { }
if (-not $version) { $version = 'dev' }
if (-not $commit) { $commit = 'none' }

$date    = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-X $Module/internal/version.Version=$version " +
           "-X $Module/internal/version.Commit=$commit " +
           "-X $Module/internal/version.BuildTime=$date"

# PowerShell 不会因为原生命令返回非 0 而抛异常，必须显式检查 $LASTEXITCODE，
# 否则「构建失败」也会打印成功提示。
function Invoke-Step {
    param([string]$Label, [scriptblock]$Action)
    & $Action
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[失败] $Label (exit $LASTEXITCODE)" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

switch ($Task) {
    'run' {
        Invoke-Step 'run' { go run ./cmd/server }
    }
    'build' {
        New-Item -ItemType Directory -Force -Path bin | Out-Null
        Invoke-Step 'build' { go build -ldflags $ldflags -o bin/leetnote-api.exe ./cmd/server }
        Write-Host "已生成 bin/leetnote-api.exe  (version=$version commit=$commit)" -ForegroundColor Green
    }
    'test' {
        Invoke-Step 'test' { go test ./... -race -cover }
    }
    'cover' {
        Invoke-Step 'cover' { go test ./... -coverprofile=coverage.out }
        go tool cover -html coverage.out
    }
    'fmt' {
        Invoke-Step 'fmt' { go fmt ./... }
        Invoke-Step 'tidy' { go mod tidy }
    }
    'tidy' {
        Invoke-Step 'tidy' { go mod tidy }
    }
    'lint' {
        Invoke-Step 'lint' { golangci-lint run ./... }
    }
    'db-init' {
        # 装扩展需要超级用户，所以这里用 postgres 账号连。
        # 不放在 migrations/ 里的原因见 scripts/init-extensions.sql 的注释。
        $pgUser = if ($env:PG_SUPERUSER) { $env:PG_SUPERUSER } else { 'postgres' }
        $pgPass = if ($env:PG_SUPERPASS) { $env:PG_SUPERPASS } else { 'postgres' }
        $env:PGPASSWORD = $pgPass
        Invoke-Step 'db-init' {
            psql -U $pgUser -h localhost -d leetnote -f scripts/init-extensions.sql
        }
        Write-Host '扩展已就绪，现在可以执行 .\dev.ps1 migrate-up' -ForegroundColor Green
    }
    'migrate-up' {
        # 用 `go run` 调项目自带的迁移子命令，而不是要求先装 golang-migrate CLI：
        # 换台机器只要有 Go 就能跑，少一个「clone 下来跑不起来」的坎。
        Invoke-Step 'migrate-up' { go run ./cmd/migrate up }
    }
    'migrate-down' {
        Invoke-Step 'migrate-down' { go run ./cmd/migrate down $Steps }
    }
    'migrate-version' {
        Invoke-Step 'migrate-version' { go run ./cmd/migrate version }
    }
    'migrate-force' {
        if ($DbVersion -lt 0) {
            Write-Host '[错误] migrate-force 需要 -DbVersion 参数，例如 .\dev.ps1 migrate-force -DbVersion 6' -ForegroundColor Red
            exit 1
        }
        Invoke-Step 'migrate-force' { go run ./cmd/migrate force $DbVersion }
    }
}
