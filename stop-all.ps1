# 停止 LeetNote 的四个服务。
#
# 用法：
#     .\stop-all.ps1             # 停全部（PostgreSQL 默认也停）
#     .\stop-all.ps1 -KeepDb     # 保留 PostgreSQL（它启动较慢，重启用得上）
#
# 注意：这里只停【本项目占用的端口】上的进程，不会误伤别的项目。

param(
    [switch]$KeepDb
)

$ports = @(
    @{ Port = 5173; Name = '前端' },
    @{ Port = 8000; Name = 'AI 服务' },
    @{ Port = 8080; Name = 'Go 服务' },
    @{ Port = 5432; Name = 'PostgreSQL' }
)

foreach ($p in $ports) {
    if ($p.Port -eq 5432 -and $KeepDb) {
        Write-Host ("  ⏭  {0,-10} :{1}（按要求保留）" -f $p.Name, $p.Port) -ForegroundColor DarkGray
        continue
    }

    $conn = Get-NetTCPConnection -LocalPort $p.Port -State Listen -EA SilentlyContinue |
        Select-Object -First 1

    if (-not $conn) {
        Write-Host ("  ⚪ {0,-10} :{1} 未运行" -f $p.Name, $p.Port) -ForegroundColor DarkGray
        continue
    }

    # PostgreSQL 用 pg_ctl 停才干净（直接杀进程下次启动可能要恢复）
    if ($p.Port -eq 5432) {
        $pgCtl = 'D:\dev\pg-stop.ps1'
        if (Test-Path $pgCtl) {
            & $pgCtl | Out-Null
            Write-Host ("  ✅ {0,-10} :{1}" -f $p.Name, $p.Port) -ForegroundColor Green
            continue
        }
    }

    $proc = Get-Process -Id $conn.OwningProcess -EA SilentlyContinue
    if ($proc) {
        # 前端是 cmd.exe 拉起来的，连子进程一起收掉，否则 5173 会留在 LISTEN
        Stop-Process -Id $proc.Id -Force -EA SilentlyContinue
        Write-Host ("  ✅ {0,-10} :{1}（pid {2}）" -f $p.Name, $p.Port, $proc.Id) -ForegroundColor Green
    }
}

# 前端用 cmd → node 两层进程，收掉 cmd 后 node 可能还占着端口，补一刀
foreach ($port in 5173, 8080, 8000) {
    $left = Get-NetTCPConnection -LocalPort $port -State Listen -EA SilentlyContinue |
        Select-Object -First 1
    if ($left) {
        Stop-Process -Id $left.OwningProcess -Force -EA SilentlyContinue
        Write-Host ("  ✅ 清理残留 :{0}" -f $port) -ForegroundColor Green
    }
}
