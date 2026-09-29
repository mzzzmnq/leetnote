# =============================================================
# LeetNote 启动器（交互式菜单）
# =============================================================
#
# 双击项目根目录的 LeetNote.bat 即可打开本启动器。
#
# 也可以直接命令行用：
#     pwsh -File launcher.ps1
#     pwsh -File launcher.ps1 -Action status    # 跑一次就退出，不进菜单
#
# 之所以做这个：每次重开机要手工敲四条命令拉起四个服务，
# 还得记住各自的启动方式（PostgreSQL 要 pg_ctl、AI 必须用 run.py、
# 前端是 cmd/npm……）。这里把那些细节都封装掉了。

param(
    [ValidateSet('menu', 'start', 'stop', 'restart', 'status', 'init')]
    [string]$Action = 'menu'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. "$PSScriptRoot\scripts\services.ps1"

$StartScript = Join-Path $PSScriptRoot 'start-all.ps1'
$StopScript = Join-Path $PSScriptRoot 'stop-all.ps1'
$ApiDevScript = Join-Path $LeetNoteRoot 'leetnote-api\dev.ps1'

# ---------------------------------------------------------------
# 排版辅助：中文字符占两格，直接 PadRight 会错位
# ---------------------------------------------------------------
function Get-StrWidth {
    param([string]$Text)
    $w = 0
    foreach ($ch in $Text.ToCharArray()) {
        $c = [int]$ch
        $wide = ($c -ge 0x1100 -and $c -le 0x115F) -or
                ($c -ge 0x2E80 -and $c -le 0xA4CF) -or
                ($c -ge 0xAC00 -and $c -le 0xD7A3) -or
                ($c -ge 0xF900 -and $c -le 0xFAFF) -or
                ($c -ge 0xFE30 -and $c -le 0xFE6F) -or
                ($c -ge 0xFF00 -and $c -le 0xFF60) -or
                ($c -ge 0xFFE0 -and $c -le 0xFFE6)
        $w += if ($wide) { 2 } else { 1 }
    }
    return $w
}

<#
清屏。
包一层 try/catch 是必要的：输出被重定向或没有真实控制台句柄时
（比如从管道调用、或某些 IDE 的集成终端），Clear-Host 会抛
"句柄无效"，把整个启动器带崩。
#>
function Clear-Screen {
    try { Clear-Host } catch { }
}

function Format-Pad {
    param([string]$Text, [int]$Width, [switch]$Right)
    $pad = [Math]::Max(0, $Width - (Get-StrWidth $Text))
    if ($Right) { return (' ' * $pad) + $Text }
    return $Text + (' ' * $pad)
}

$BOX_WIDTH = 60

function Write-BoxTop {
    Write-Host ('╭' + ('─' * $BOX_WIDTH) + '╮') -ForegroundColor DarkCyan
}
function Write-BoxBottom {
    Write-Host ('╰' + ('─' * $BOX_WIDTH) + '╯') -ForegroundColor DarkCyan
}
function Write-BoxLine {
    param([string]$Left, [string]$Right = '')
    if ($Right) {
        $inner = $BOX_WIDTH - 2
        $gap = $inner - (Get-StrWidth $Left) - (Get-StrWidth $Right)
        $line = ' ' + $Left + (' ' * [Math]::Max(1, $gap)) + $Right + ' '
    }
    else {
        $line = ' ' + (Format-Pad $Left ($BOX_WIDTH - 1))
    }
    Write-Host ('│' + $line + '│') -ForegroundColor DarkCyan
}

# ---------------------------------------------------------------
# 渲染
# ---------------------------------------------------------------
function Show-Dashboard {
    param([switch]$SkipHealth)

    Clear-Screen

    Write-Host ''
    Write-BoxTop
    Write-BoxLine 'LeetNote · 算法练习笔记' (Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
    Write-BoxBottom
    Write-Host ''

    $status = Get-LeetNoteStatus -SkipHealthCheck:$SkipHealth
    $running = 0
    Write-Host ('  ' + (Format-Pad '服务' 14) + (Format-Pad '端口' 8) + (Format-Pad '状态' 26) + 'PID') -ForegroundColor DarkGray
    Write-Host ('  ' + ('·' * 56)) -ForegroundColor DarkGray

    foreach ($s in $status) {
        if ($s.Running) { $running++ }

        $icon = if (-not $s.Running) { '❌' } elseif ($s.Health -eq $false) { '⚠️ ' } else { '✅' }
        $color = if (-not $s.Running) { 'DarkGray' } elseif ($s.Health -eq $false) { 'Yellow' } else { 'Green' }
        $pidText = if ($s.Pid) { "$($s.Pid)" } else { '-' }

        $nameCell = Format-Pad $s.Name 12
        $portCell = Format-Pad ":$($s.Port)" 8
        $stateCell = Format-Pad $s.HealthText 26

        Write-Host "  $icon " -NoNewline
        Write-Host $nameCell -NoNewline -ForegroundColor $color
        Write-Host $portCell -NoNewline -ForegroundColor DarkGray
        Write-Host $stateCell -NoNewline -ForegroundColor $color
        Write-Host $pidText -ForegroundColor DarkGray
    }

    Write-Host ''
    if ($running -eq $status.Count) {
        Write-Host "  🌐 打开 $LeetNoteWebUrl" -ForegroundColor Cyan
    }
    else {
        Write-Host "  $running / $($status.Count) 个服务在运行" -ForegroundColor Yellow
    }

    # 新机器上最容易卡住的是「工具没装 / 不在 PATH」，直接把探测结果摆出来
    $toolchain = Get-LeetNoteToolchainStatus
    if (-not $toolchain.HasGo -or -not $toolchain.HasPg) {
        Write-Host ''
        Write-Host '  ⚠️  工具链不完整：' -ForegroundColor Yellow
        if (-not $toolchain.HasGo) { Write-Host '       找不到 Go —— 请安装并加入 PATH' -ForegroundColor Yellow }
        if (-not $toolchain.HasPg) { Write-Host '       找不到 PostgreSQL —— 请安装并加入 PATH' -ForegroundColor Yellow }
        Write-Host '       也可以复制 local.config.example.ps1 为 local.config.ps1 手工指定路径' -ForegroundColor DarkGray
    }

    Write-Host ''
}

function Show-Menu {
    Write-Host '  ────────────────────────────────────────────────────────' -ForegroundColor DarkGray
    Write-Host '    [1] 启动全部      [2] 停止全部      [3] 重启全部' -ForegroundColor White
    Write-Host '    [4] 刷新状态      [5] 打开网页      [6] 查看日志' -ForegroundColor White
    Write-Host '    [7] 首次初始化                              [0] 退出' -ForegroundColor White
    Write-Host '  ────────────────────────────────────────────────────────' -ForegroundColor DarkGray
    Write-Host ''
}

# ---------------------------------------------------------------
# 动作
# ---------------------------------------------------------------
function Invoke-StartAll {
    Write-Host ''
    Write-Host '  正在启动...' -ForegroundColor Cyan
    & $StartScript
    Write-Host ''
    Write-Host '  按回车返回菜单' -ForegroundColor DarkGray
    Read-Host | Out-Null
}

function Invoke-StopAll {
    Write-Host ''
    $answer = Read-Host '  确定停止全部服务？(y/N)'
    if ($answer -notmatch '^[Yy]') { return }

    Write-Host ''
    & $StopScript
    Write-Host ''
    Write-Host '  按回车返回菜单' -ForegroundColor DarkGray
    Read-Host | Out-Null
}

function Invoke-RestartAll {
    Write-Host ''
    Write-Host '  正在重启...' -ForegroundColor Cyan
    & $StartScript -Restart
    Write-Host ''
    Write-Host '  按回车返回菜单' -ForegroundColor DarkGray
    Read-Host | Out-Null
}

function Invoke-OpenWeb {
    $status = Get-LeetNoteStatus -SkipHealthCheck
    $web = $status | Where-Object Key -eq 'web'

    if (-not $web.Running) {
        Write-Host ''
        Write-Host '  前端还没启动，先启动它？(Y/n) ' -NoNewline -ForegroundColor Yellow
        $answer = Read-Host
        if ($answer -match '^[Nn]') { return }

        & $StartScript -Only web -Quiet | Out-Null
        Start-Sleep -Milliseconds 800
    }

    Write-Host ''
    Write-Host "  正在打开 $LeetNoteWebUrl ..." -ForegroundColor Cyan
    Start-Process $LeetNoteWebUrl

    Start-Sleep -Seconds 1
}

function Invoke-ViewLog {
    Write-Host ''
    Write-Host '  查看哪个服务的日志？' -ForegroundColor Cyan
    Write-Host '    [1] Go 服务   [2] AI 服务   [3] 前端   [4] 数据库' -ForegroundColor White
    Write-Host ''

    $pick = Read-Host '  选择'
    $key = switch ($pick) {
        '1' { 'api' }
        '2' { 'ai' }
        '3' { 'web' }
        '4' { 'db' }
        default { return }
    }

    $path = Get-LeetNoteLogPath -Key $key
    # 数据库的日志不在 %TEMP%，而是 PostgreSQL 自己的数据目录下
    if ($key -eq 'db' -and $LeetNotePgData) { $path = Join-Path $LeetNotePgData 'server.log' }

    if (-not (Test-Path $path)) {
        Write-Host ''
        Write-Host "  日志文件还不存在：$path" -ForegroundColor Yellow
        Write-Host '  （这个服务可能还没被启动过）' -ForegroundColor DarkGray
        Write-Host ''
        Write-Host '  按回车返回菜单' -ForegroundColor DarkGray
        Read-Host | Out-Null
        return
    }

    Clear-Screen
    Write-Host "  实时日志：$path" -ForegroundColor Cyan
    Write-Host '  按 Ctrl+C 返回菜单' -ForegroundColor DarkGray
    Write-Host ''

    try {
        # -Wait 会一直挂着输出新内容，直到 Ctrl+C
        Get-Content -Path $path -Tail 40 -Wait
    }
    catch {
        # Ctrl+C 会让 cmdlet 抛 PipelineStoppedException，吞掉即可回到菜单
    }
}

function Invoke-FirstRunInit {
    Write-Host ''
    Write-Host '  首次初始化会做两件事：' -ForegroundColor Cyan
    Write-Host '    ① 用超级用户安装数据库扩展（pg_trgm / pgcrypto / vector）' -ForegroundColor Gray
    Write-Host '    ② 用应用账号执行 6 个版本化迁移建表' -ForegroundColor Gray
    Write-Host ''
    Write-Host '  已经有数据的环境不需要执行，重复执行也不会丢数据。' -ForegroundColor DarkGray
    Write-Host ''
    $answer = Read-Host '  继续？(y/N)'
    if ($answer -notmatch '^[Yy]') { return }

    Write-Host ''
    if (-not (Test-Path $ApiDevScript)) {
        Write-Host "  ❌ 找不到 $ApiDevScript" -ForegroundColor Red
        Read-Host '  按回车返回菜单' | Out-Null
        return
    }

    Push-Location (Join-Path $LeetNoteRoot 'leetnote-api')
    try {
        & $ApiDevScript db-init
        & $ApiDevScript migrate-up
    }
    finally { Pop-Location }

    Write-Host ''
    Write-Host '  按回车返回菜单' -ForegroundColor DarkGray
    Read-Host | Out-Null
}

# ---------------------------------------------------------------
# 非交互模式
# ---------------------------------------------------------------
if ($Action -ne 'menu') {
    switch ($Action) {
        'start' { & $StartScript }
        'stop' { & $StopScript }
        'restart' { & $StartScript -Restart }
        'status' { Show-Dashboard }
        'init' { Invoke-FirstRunInit }
    }
    exit 0
}

# ---------------------------------------------------------------
# 主循环
# ---------------------------------------------------------------
while ($true) {
    Show-Dashboard
    Show-Menu

    $choice = Read-Host '  请选择'

    switch ($choice) {
        '1' { Invoke-StartAll }
        '2' { Invoke-StopAll }
        '3' { Invoke-RestartAll }
        '4' { }                    # 什么都不做，下一轮循环会刷新
        '5' { Invoke-OpenWeb }
        '6' { Invoke-ViewLog }
        '7' { Invoke-FirstRunInit }
        '0' { break }
        default { }                # 无效输入忽略，直接刷新
    }

    if ($choice -eq '0') { break }
}

Clear-Screen
Write-Host ''
Write-Host '  已退出 LeetNote 启动器。' -ForegroundColor DarkGray
Write-Host '  （服务仍在后台运行，需要停止请再打开一次选 [2]）' -ForegroundColor DarkGray
Write-Host ''
