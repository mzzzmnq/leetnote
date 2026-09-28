# =============================================================
# 查看 LeetNote 服务状态
# =============================================================
#
# 用法：
#     .\status.ps1
#     .\status.ps1 -NoHealth    # 只探端口，不发健康请求（更快）
#
# 服务清单在 scripts/services.ps1 里统一定义。

param(
    [switch]$NoHealth,
    [switch]$Quiet
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. "$PSScriptRoot\scripts\services.ps1"

$status = Get-LeetNoteStatus -SkipHealthCheck:$NoHealth

if (-not $Quiet) {
    Write-Host ''
    Write-Host 'LeetNote 服务状态' -ForegroundColor Cyan
    Write-Host ('─' * 62) -ForegroundColor DarkGray
}

$down = 0
foreach ($s in $status) {
    if (-not $s.Running) {
        Write-Host ("  ❌ {0,-11} :{1,-5} 未运行" -f $s.Name, $s.Port) -ForegroundColor Red
        $down++
        continue
    }

    $color = if ($s.Health -eq $false) { 'Yellow' } else { 'Green' }
    Write-Host ("  ✅ {0,-11} :{1,-5} pid={2,-7} {3}" -f $s.Name, $s.Port, $s.Pid, $s.HealthText) -ForegroundColor $color
}

if (-not $Quiet) {
    Write-Host ('─' * 62) -ForegroundColor DarkGray
    if ($down -eq 0) {
        Write-Host "  全部正常，打开 $LeetNoteWebUrl" -ForegroundColor Green
    }
    else {
        Write-Host "  有 $down 个服务未运行，执行 .\start-all.ps1 启动" -ForegroundColor Yellow
    }
    Write-Host ''
}

# 返回退出码：全部正常 0，否则 1（方便脚本里判断）
exit $(if ($down -eq 0) { 0 } else { 1 })
