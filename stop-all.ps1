# =============================================================
# 停止 LeetNote 的服务
# =============================================================
#
# 用法：
#     .\stop-all.ps1             # 停全部（PostgreSQL 也停）
#     .\stop-all.ps1 -KeepDb     # 保留 PostgreSQL（它启动较慢，重启用得上）
#     .\stop-all.ps1 -Quiet      # 少打印（给启动器调用）
#
# 只停【本项目占用的端口】上的进程，不会误伤别的项目。
# 服务清单在 scripts/services.ps1 里统一定义。

param(
    [switch]$KeepDb,
    [switch]$Quiet
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. "$PSScriptRoot\scripts\services.ps1"

# 停止顺序：先停应用，最后停数据库（应用可能还在写库）
$stopOrder = @('web', 'ai', 'api', 'db')

foreach ($key in $stopOrder) {
    $svc = Get-LeetNoteServices | Where-Object Key -eq $key

    if ($key -eq 'db' -and $KeepDb) {
        if (-not $Quiet) {
            Write-Host ("  ⏭  {0,-11} :{1}（按要求保留）" -f $svc.Name, $svc.Port) -ForegroundColor DarkGray
        }
        continue
    }

    $conn = Get-NetTCPConnection -LocalPort $svc.Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -First 1

    if (-not $conn) {
        if (-not $Quiet) {
            Write-Host ("  ⚪ {0,-11} :{1} 未运行" -f $svc.Name, $svc.Port) -ForegroundColor DarkGray
        }
        continue
    }

    # PostgreSQL 用 pg_ctl 停才干净（直接杀进程下次启动可能要恢复）
    if ($key -eq 'db') {
        $pgStop = 'D:\dev\pg-stop.ps1'
        if (Test-Path $pgStop) {
            & $pgStop *> $null
            Write-Host ("  ✅ {0,-11} :{1}" -f $svc.Name, $svc.Port) -ForegroundColor Green
            continue
        }
    }

    Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
    Write-Host ("  ✅ {0,-11} :{1}（pid {2}）" -f $svc.Name, $svc.Port, $conn.OwningProcess) -ForegroundColor Green
}

# 前端是 cmd.exe → node 两层进程，收掉 cmd 后 node 可能还占着端口，补一刀。
# 稍微等一下再检查，否则进程还没来得及释放端口。
Start-Sleep -Milliseconds 600

foreach ($key in $stopOrder) {
    if ($key -eq 'db' -and $KeepDb) { continue }

    $svc = Get-LeetNoteServices | Where-Object Key -eq $key
    $left = Get-NetTCPConnection -LocalPort $svc.Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -First 1

    if ($left) {
        Stop-Process -Id $left.OwningProcess -Force -ErrorAction SilentlyContinue
        Write-Host ("  ✅ 清理残留 {0} :{1}" -f $svc.Name, $svc.Port) -ForegroundColor Green
    }
}
