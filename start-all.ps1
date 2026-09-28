# =============================================================
# 一键启动 LeetNote 的四个服务
# =============================================================
#
# 用法：
#     .\start-all.ps1            # 启动全部
#     .\start-all.ps1 -Only api  # 只启动某一个（db / api / ai / web）
#     .\start-all.ps1 -Restart   # 先停掉再启动
#
# ---------------------------------------------------------------
# 【为什么必须用 Start-Process】
#
# 直接在 shell 里跑 `go run ./cmd/server`，进程会成为当前 shell 的子进程，
# 一旦这个 shell 退出（命令超时、关窗口、编辑器重启），服务就被一起带走。
#
# Start-Process 会让进程脱离当前 shell 独立存活 —— 这正是
# leetnote-api 里 pg-start.ps1 注释提到的那类进程上下文问题。
#
# 日志统一写到 $env:TEMP\leetnote-logs\，方便排查。
# =============================================================

param(
    [ValidateSet('all', 'db', 'api', 'ai', 'web')]
    [string]$Only = 'all',

    # 先停掉已在运行的服务再启动
    [switch]$Restart
)

$ErrorActionPreference = 'Stop'

$Root    = $PSScriptRoot
$ApiDir  = Join-Path $Root 'leetnote-api'
$AiDir   = Join-Path $Root 'leetnote-ai'
$WebDir  = Join-Path $Root 'frontend'
$LogDir  = Join-Path $env:TEMP 'leetnote-logs'

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

if ($Restart) {
    & (Join-Path $Root 'stop-all.ps1') | Out-Null
    Start-Sleep -Seconds 1
}

function Test-Port([int]$Port) {
    return [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -EA SilentlyContinue)
}

function Wait-Port([int]$Port, [string]$Name, [int]$Seconds = 45) {
    for ($i = 0; $i -lt ($Seconds * 2); $i++) {
        if (Test-Port $Port) {
            Write-Host ("  ✅ {0,-10} :{1}" -f $Name, $Port) -ForegroundColor Green
            return $true
        }
        Start-Sleep -Milliseconds 500
    }
    Write-Host ("  ❌ {0,-10} :{1} 启动超时" -f $Name, $Port) -ForegroundColor Red
    return $false
}

function Should-Run([string]$Name) {
    return ($Only -eq 'all' -or $Only -eq $Name)
}

# ---------------------------------------------------------------- 1. PostgreSQL
if (Should-Run 'db') {
    if (Test-Port 5432) {
        Write-Host '  ✅ PostgreSQL  :5432（已在运行）' -ForegroundColor Green
    } else {
        $starter = 'D:\dev\pg-start.ps1'
        if (Test-Path $starter) {
            & $starter | Out-Null
            Wait-Port 5432 'PostgreSQL' | Out-Null
        } else {
            Write-Host "  ⚠️  找不到 $starter，请手工启动 PostgreSQL" -ForegroundColor Yellow
        }
    }
}

# ---------------------------------------------------------------- 2. Go 主服务
if (Should-Run 'api') {
    if (Test-Port 8080) {
        Write-Host '  ✅ Go 服务    :8080（已在运行）' -ForegroundColor Green
    } else {
        # 优先用编译好的二进制；没有就先构建（比 go run 启动快且稳定）
        $exe = Join-Path $ApiDir 'bin\leetnote-api.exe'
        if (-not (Test-Path $exe)) {
            Write-Host '  🔨 首次运行，正在构建 Go 服务...' -ForegroundColor Cyan
            $env:GOPROXY = if ($env:GOPROXY) { $env:GOPROXY } else { 'https://goproxy.cn,direct' }
            Push-Location $ApiDir
            go build -o bin/leetnote-api.exe ./cmd/server
            Pop-Location
        }

        Start-Process -FilePath $exe `
            -WorkingDirectory $ApiDir `
            -RedirectStandardOutput (Join-Path $LogDir 'api.out.log') `
            -RedirectStandardError  (Join-Path $LogDir 'api.err.log') `
            -WindowStyle Hidden

        Wait-Port 8080 'Go 服务' | Out-Null
    }
}

# ---------------------------------------------------------------- 3. AI 服务
if (Should-Run 'ai') {
    if (Test-Port 8000) {
        Write-Host '  ✅ AI 服务    :8000（已在运行）' -ForegroundColor Green
    } else {
        # 【只能用 venv 里的 python】不是 run.py 的问题，是这个组合才配了
        # 正确的 OpenSSL（详见 leetnote-ai/README 与 docs/ENVIRONMENT.md）
        $py = Join-Path $AiDir '.venv\Scripts\python.exe'
        if (-not (Test-Path $py)) {
            Write-Host "  ⚠️  找不到 $py，跳过 AI 服务" -ForegroundColor Yellow
            Write-Host '      首次使用请按 leetnote-ai/README 建虚拟环境' -ForegroundColor Yellow
        } else {
            # 必须用 run.py 启动，不能直接 uvicorn（事件循环不兼容，见 run.py 注释）
            Start-Process -FilePath $py `
                -ArgumentList 'run.py' `
                -WorkingDirectory $AiDir `
                -RedirectStandardOutput (Join-Path $LogDir 'ai.out.log') `
                -RedirectStandardError  (Join-Path $LogDir 'ai.err.log') `
                -WindowStyle Hidden

            Wait-Port 8000 'AI 服务' | Out-Null
        }
    }
}

# ---------------------------------------------------------------- 4. 前端
if (Should-Run 'web') {
    if (Test-Port 5173) {
        Write-Host '  ✅ 前端       :5173（已在运行）' -ForegroundColor Green
    } else {
        Start-Process -FilePath 'cmd.exe' `
            -ArgumentList '/c', 'npm', 'run', 'dev' `
            -WorkingDirectory $WebDir `
            -RedirectStandardOutput (Join-Path $LogDir 'web.out.log') `
            -RedirectStandardError  (Join-Path $LogDir 'web.err.log') `
            -WindowStyle Hidden

        Wait-Port 5173 '前端' | Out-Null
    }
}

Write-Host ''
Write-Host '打开 http://localhost:5173' -ForegroundColor Cyan
Write-Host "日志目录：$LogDir" -ForegroundColor DarkGray
