# =============================================================
# 一键启动 LeetNote 的全部服务
# =============================================================
#
# 用法：
#     .\start-all.ps1              # 启动全部
#     .\start-all.ps1 -Only api    # 只启动某一个（db / api / ai / web）
#     .\start-all.ps1 -Restart     # 先停掉再启动
#     .\start-all.ps1 -Quiet       # 少打印（给启动器调用）
#
# 服务清单在 scripts/services.ps1 里统一定义。
#
# ---------------------------------------------------------------
# 【为什么必须用 Start-Process】
#
# 直接在 shell 里跑 `go run ./cmd/server`，进程会成为当前 shell 的子进程，
# 一旦这个 shell 退出（命令超时、关窗口、编辑器重启、机器重启），
# 服务就被一起带走。
#
# Start-Process 会让进程脱离当前 shell 独立存活。
#
# 日志统一写到 $env:TEMP\leetnote-logs\，方便排查。
# =============================================================

param(
    [ValidateSet('all', 'db', 'api', 'ai', 'web')]
    [string]$Only = 'all',

    # 先停掉已在运行的服务再启动
    [switch]$Restart,

    # 只输出错误和结果行（启动器调用时用）
    [switch]$Quiet
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. "$PSScriptRoot\scripts\services.ps1"

New-Item -ItemType Directory -Force -Path $LeetNoteLogDir | Out-Null

function Write-Info([string]$Text) { if (-not $Quiet) { Write-Host $Text -ForegroundColor Cyan } }

if ($Restart) {
    & (Join-Path $PSScriptRoot 'stop-all.ps1') -Quiet | Out-Null
    Start-Sleep -Seconds 1
}

function Should-Run([string]$Key) {
    return ($Only -eq 'all' -or $Only -eq $Key)
}

# 轮询等待端口就绪
function Wait-LeetNotePort {
    param(
        [Parameter(Mandatory)][int]$Port,
        [Parameter(Mandatory)][string]$Name,
        [int]$TimeoutSeconds = 45
    )

    for ($i = 0; $i -lt ($TimeoutSeconds * 2); $i++) {
        if (Test-LeetNotePort -Port $Port) {
            Write-Host ("  ✅ {0,-11} :{1}" -f $Name, $Port) -ForegroundColor Green
            return $true
        }
        Start-Sleep -Milliseconds 500
    }

    Write-Host ("  ❌ {0,-11} :{1} 启动超时" -f $Name, $Port) -ForegroundColor Red
    return $false
}

# ---------------------------------------------------------------- PostgreSQL
if (Should-Run 'db') {
    $svc = (Get-LeetNoteServices | Where-Object Key -eq 'db')

    if (Test-LeetNotePort -Port $svc.Port) {
        Write-Host ("  ✅ {0,-11} :{1}（已在运行）" -f $svc.Name, $svc.Port) -ForegroundColor Green
    }
    else {
        # 便携版 PostgreSQL 有自己的启动脚本（含残留 pid 清理、进程上下文处理）
        $starter = 'D:\dev\pg-start.ps1'
        if (Test-Path $starter) {
            & $starter *> $null
            Wait-LeetNotePort -Port $svc.Port -Name $svc.Name | Out-Null
        }
        else {
            Write-Host "  ⚠️  找不到 $starter，请手工启动 PostgreSQL" -ForegroundColor Yellow
        }
    }
}

# ---------------------------------------------------------------- Go 主服务
if (Should-Run 'api') {
    $svc = (Get-LeetNoteServices | Where-Object Key -eq 'api')
    $apiDir = Join-Path $LeetNoteRoot 'leetnote-api'

    if (Test-LeetNotePort -Port $svc.Port) {
        Write-Host ("  ✅ {0,-11} :{1}（已在运行）" -f $svc.Name, $svc.Port) -ForegroundColor Green
    }
    else {
        # 优先用编译好的二进制；没有就先构建（比 go run 启动快且更稳）
        $exe = Join-Path $apiDir 'bin\leetnote-api.exe'
        if (-not (Test-Path $exe)) {
            Write-Info '  🔨 首次运行，正在构建 Go 服务...'
            if (-not $env:GOPROXY) { $env:GOPROXY = 'https://goproxy.cn,direct' }
            Push-Location $apiDir
            try { go build -o bin/leetnote-api.exe ./cmd/server }
            finally { Pop-Location }
        }

        Start-Process -FilePath $exe `
            -WorkingDirectory $apiDir `
            -RedirectStandardOutput (Get-LeetNoteLogPath -Key 'api') `
            -RedirectStandardError (Join-Path $LeetNoteLogDir 'api.err.log') `
            -WindowStyle Hidden

        Wait-LeetNotePort -Port $svc.Port -Name $svc.Name | Out-Null
    }
}

# ---------------------------------------------------------------- AI 服务
if (Should-Run 'ai') {
    $svc = (Get-LeetNoteServices | Where-Object Key -eq 'ai')
    $aiDir = Join-Path $LeetNoteRoot 'leetnote-ai'

    if (Test-LeetNotePort -Port $svc.Port) {
        Write-Host ("  ✅ {0,-11} :{1}（已在运行）" -f $svc.Name, $svc.Port) -ForegroundColor Green
    }
    else {
        # 【只能用 venv 里的 python】这个组合才配了正确的 OpenSSL
        # （Anaconda 的 Python 会导致所有 HTTPS 请求失败，见 docs/ENVIRONMENT.md）
        $py = Join-Path $aiDir '.venv\Scripts\python.exe'
        if (-not (Test-Path $py)) {
            Write-Host "  ⚠️  找不到 $py，跳过 AI 服务" -ForegroundColor Yellow
            Write-Host '      首次使用请按 leetnote-ai/README 建虚拟环境' -ForegroundColor Yellow
        }
        else {
            # 必须用 run.py 启动，不能直接 uvicorn（事件循环不兼容，见 run.py 注释）
            Start-Process -FilePath $py `
                -ArgumentList 'run.py' `
                -WorkingDirectory $aiDir `
                -RedirectStandardOutput (Get-LeetNoteLogPath -Key 'ai') `
                -RedirectStandardError (Join-Path $LeetNoteLogDir 'ai.err.log') `
                -WindowStyle Hidden

            Wait-LeetNotePort -Port $svc.Port -Name $svc.Name | Out-Null
        }
    }
}

# ---------------------------------------------------------------- 前端
if (Should-Run 'web') {
    $svc = (Get-LeetNoteServices | Where-Object Key -eq 'web')
    $webDir = Join-Path $LeetNoteRoot 'frontend'

    if (Test-LeetNotePort -Port $svc.Port) {
        Write-Host ("  ✅ {0,-11} :{1}（已在运行）" -f $svc.Name, $svc.Port) -ForegroundColor Green
    }
    else {
        Start-Process -FilePath 'cmd.exe' `
            -ArgumentList '/c', 'npm', 'run', 'dev' `
            -WorkingDirectory $webDir `
            -RedirectStandardOutput (Get-LeetNoteLogPath -Key 'web') `
            -RedirectStandardError (Join-Path $LeetNoteLogDir 'web.err.log') `
            -WindowStyle Hidden

        Wait-LeetNotePort -Port $svc.Port -Name $svc.Name | Out-Null
    }
}

if (-not $Quiet) {
    Write-Host ''
    Write-Host "打开 $LeetNoteWebUrl" -ForegroundColor Cyan
    Write-Host "日志目录：$LeetNoteLogDir" -ForegroundColor DarkGray
}
