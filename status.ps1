# 查看四个服务的运行状态与健康检查。
#
# 用法：.\status.ps1

$services = @(
    @{ Port = 5432; Name = 'PostgreSQL'; Url = $null },
    @{ Port = 8000; Name = 'AI 服务';    Url = 'http://127.0.0.1:8000/health' },
    @{ Port = 8080; Name = 'Go 服务';    Url = 'http://127.0.0.1:8080/health' },
    @{ Port = 5173; Name = '前端';       Url = 'http://127.0.0.1:5173/' }
)

Write-Host ''
Write-Host 'LeetNote 服务状态' -ForegroundColor Cyan
Write-Host ('─' * 58) -ForegroundColor DarkGray

$down = 0
foreach ($s in $services) {
    $conn = Get-NetTCPConnection -LocalPort $s.Port -State Listen -EA SilentlyContinue |
        Select-Object -First 1

    if (-not $conn) {
        Write-Host ("  ❌ {0,-11} :{1,-5} 未运行" -f $s.Name, $s.Port) -ForegroundColor Red
        $down++
        continue
    }

    $health = ''
    if ($s.Url) {
        try {
            $resp = Invoke-WebRequest -Uri $s.Url -TimeoutSec 5 -UseBasicParsing
            $health = "HTTP $($resp.StatusCode)"
        } catch {
            # 端口在监听但请求失败 —— 通常还在启动中
            $health = '端口在监听，但请求失败'
        }
    } else {
        $health = 'accepting connections'
    }

    Write-Host ("  ✅ {0,-11} :{1,-5} pid={2,-7} {3}" -f $s.Name, $s.Port, $conn.OwningProcess, $health) -ForegroundColor Green
}

Write-Host ('─' * 58) -ForegroundColor DarkGray
if ($down -eq 0) {
    Write-Host '  全部正常，打开 http://localhost:5173' -ForegroundColor Green
} else {
    Write-Host "  有 $down 个服务未运行，执行 .\start-all.ps1 启动" -ForegroundColor Yellow
}
Write-Host ''
