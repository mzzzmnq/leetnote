# Windows 下的开发任务脚本（替代 Makefile，因为本机没装 make）
# 用法：.\dev.ps1 [run|build|test|cover|fmt|tidy|lint|migrate-up|migrate-down]
param(
    [Parameter(Position = 0)]
    [ValidateSet('run', 'build', 'test', 'cover', 'fmt', 'tidy', 'lint', 'migrate-up', 'migrate-down')]
    [string]$Task = 'run'
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
$DbUrl  = 'postgres://leetnote:leetnote@localhost:5432/leetnote?sslmode=disable'

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
    'migrate-up' {
        Invoke-Step 'migrate-up' { migrate -path migrations -database $DbUrl up }
    }
    'migrate-down' {
        Invoke-Step 'migrate-down' { migrate -path migrations -database $DbUrl down 1 }
    }
}
