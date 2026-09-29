# =============================================================
# PostgreSQL 启停（自包含，不依赖仓库外的脚本）
# =============================================================
#
# 原先 start-all.ps1 调用的是 D:\dev\pg-start.ps1 —— 那是**仓库外**的文件，
# 换台电脑根本不会跟过去。现在把逻辑收进项目里，路径由 local-paths.ps1 探测。
#
# 提供：
#     Start-LeetNotePostgres
#     Stop-LeetNotePostgres
#     Test-LeetNotePostgresReady
# =============================================================

Set-StrictMode -Version Latest

. "$PSScriptRoot\local-paths.ps1"

<#
判断数据库是否已经在接受连接。
用 pg_isready 而不是探端口：端口开着不代表已经能处理查询
（PostgreSQL 启动过程中会先监听端口，但还没准备好接客）。
#>
function Test-LeetNotePostgresReady {
    if (-not $LeetNotePgBin) { return $false }
    $isReady = Join-Path $LeetNotePgBin 'pg_isready.exe'
    if (-not (Test-Path $isReady)) { return $false }

    & $isReady -h $LeetNoteDbHost -p $LeetNoteDbPort *> $null
    return $LASTEXITCODE -eq 0
}

<#
启动 PostgreSQL。

【必须用 Start-Process 让 pg_ctl 完全脱离当前 shell】

踩过的坑：如果在内联执行 `pg_ctl start`，一旦调用方 shell 被强杀
（命令超时、关闭窗口、编辑器重启），postmaster 会存活在一个**受限的进程上下文**里，
之后无法创建后端进程 —— 每次客户端连接都报：

    client backend was terminated by exception 0xC0000142

0xC0000142 = STATUS_DLL_INIT_FAILED，很容易误判成杀毒软件拦截。
实际是 postmaster 的进程上下文坏了，只能重启数据库恢复。

另外 `pg_ctl -w`（等待）也有风险：它会一直占着调用方的 shell。
所以这里用 Start-Process 分离 + 自己轮询端口。
#>
function Start-LeetNotePostgres {
    param([int]$TimeoutSeconds = 20)

    if (Test-LeetNotePostgresReady) {
        return $true
    }

    if (-not $LeetNotePgData) {
        throw "找不到 PostgreSQL 数据目录。请在 local.config.ps1 里设置 `$LeetNotePgData。"
    }

    $pgCtl = Get-LeetNotePgExe 'pg_ctl.exe'
    $logFile = Join-Path $LeetNotePgData 'server.log'

    # ---------- 清理崩溃残留 ----------
    # postmaster.pid 还在但进程已经没了 → 删掉，否则 pg_ctl 会拒绝启动
    $pidFile = Join-Path $LeetNotePgData 'postmaster.pid'
    if (Test-Path $pidFile) {
        $stalePid = Get-Content $pidFile -TotalCount 1 -ErrorAction SilentlyContinue
        if ($stalePid -and -not (Get-Process -Id $stalePid -ErrorAction SilentlyContinue)) {
            Write-Host "  清理残留的 postmaster.pid（PID $stalePid 已不存在）" -ForegroundColor DarkGray
            Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
        }
    }

    # ---------- 分离启动 ----------
    Start-Process -FilePath $pgCtl `
        -ArgumentList '-D', $LeetNotePgData, '-l', $logFile, 'start' `
        -WindowStyle Hidden

    # ---------- 轮询等待就绪 ----------
    for ($i = 0; $i -lt ($TimeoutSeconds * 2); $i++) {
        Start-Sleep -Milliseconds 500
        if (Test-LeetNotePostgresReady) { return $true }
    }

    Write-Host "  PostgreSQL 启动超时。日志：$logFile" -ForegroundColor Red
    Write-Host "  若日志里出现 0xC0000142，说明 postmaster 上下文损坏，执行：" -ForegroundColor Yellow
    Write-Host "    Get-Process postgres | Stop-Process -Force" -ForegroundColor Yellow
    Write-Host "  然后重试。" -ForegroundColor Yellow
    return $false
}

<#
停止 PostgreSQL。

用 pg_ctl stop 而不是直接杀进程 —— 前者会走完整的 shutdown 流程
（刷盘、清理共享内存），后者可能让下次启动需要做崩溃恢复。
#>
function Stop-LeetNotePostgres {
    param([int]$TimeoutSeconds = 20)

    if (-not (Test-LeetNotePostgresReady)) {
        return $true
    }
    if (-not $LeetNotePgData) {
        return $false
    }

    $pgCtl = Get-LeetNotePgExe 'pg_ctl.exe'

    Start-Process -FilePath $pgCtl `
        -ArgumentList '-D', $LeetNotePgData, '-m', 'fast', 'stop' `
        -WindowStyle Hidden

    for ($i = 0; $i -lt ($TimeoutSeconds * 2); $i++) {
        Start-Sleep -Milliseconds 500
        if (-not (Test-LeetNotePostgresReady)) { return $true }
    }

    Write-Host '  PostgreSQL 停止超时' -ForegroundColor Yellow
    return $false
}
