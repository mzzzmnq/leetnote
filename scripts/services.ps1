# =============================================================
# LeetNote 服务清单 —— 单一事实来源
# =============================================================
#
# 之前 start-all / stop-all / status 三个脚本各自写了一遍端口和名称，
# 加一个服务（比如以后上 Redis）要改三四个地方，很容易漏。
# 现在统一在这里定义，其它脚本 dot-source 本文件即可。
#
# 用法（在其它脚本里）：
#     . "$PSScriptRoot\scripts\services.ps1"
#     $services = Get-LeetNoteServices

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# 项目根目录（本文件在 <root>\scripts\ 下）
$LeetNoteRoot = Split-Path -Parent $PSScriptRoot

# 日志目录：所有服务的 stdout/stderr 都落在这里
$LeetNoteLogDir = Join-Path $env:TEMP 'leetnote-logs'

<#
服务定义。
  Key      脚本里引用它的短名（-Only db 之类）
  Name     显示名
  Port     监听端口，也是"是否在运行"的判定依据
  Health   健康检查地址；为 $null 表示只探端口
  Desc     一句话说明，给启动器显示
#>
function Get-LeetNoteServices {
    @(
        [pscustomobject]@{
            Key = 'db'; Name = 'PostgreSQL'; Port = 5432; Desc = '数据库'
            Health = $null
        }
        [pscustomobject]@{
            Key = 'api'; Name = 'Go 服务'; Port = 8080; Desc = '主服务（HTTP API）'
            Health = 'http://127.0.0.1:8080/health'
        }
        [pscustomobject]@{
            Key = 'ai'; Name = 'AI 服务'; Port = 8000; Desc = '向量检索 + LLM'
            Health = 'http://127.0.0.1:8000/health'
        }
        [pscustomobject]@{
            Key = 'web'; Name = '前端'; Port = 5173; Desc = 'Vue 3 开发服务器'
            Health = 'http://127.0.0.1:5173/'
        }
    )
}

# 前端访问地址
$LeetNoteWebUrl = 'http://localhost:5173'

<#
判断端口是否在监听。
用 Get-NetTCPConnection 而不是 Test-NetConnection —— 后者在某些
Windows 版本上要等好几秒才返回，启动器里轮询会非常卡。
#>
function Test-LeetNotePort {
    param([Parameter(Mandatory)][int]$Port)

    return [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
        Select-Object -First 1)
}

<#
采集每个服务的当前状态。
返回对象数组，字段：Key / Name / Port / Desc / Running / Pid / Health / HealthText
#>
function Get-LeetNoteStatus {
    param(
        [switch]$SkipHealthCheck
    )

    $result = @()

    foreach ($svc in Get-LeetNoteServices) {
        $conn = Get-NetTCPConnection -LocalPort $svc.Port -State Listen -ErrorAction SilentlyContinue |
            Select-Object -First 1

        $healthOk = $null
        $healthText = ''

        if (-not $conn) {
            $healthText = '未运行'
        }
        elseif ($SkipHealthCheck -or -not $svc.Health) {
            $healthText = if ($svc.Health) { '(未检查)' } else { 'accepting connections' }
        }
        else {
            try {
                $resp = Invoke-WebRequest -Uri $svc.Health -TimeoutSec 5 -UseBasicParsing
                $healthOk = $true
                $healthText = "HTTP $($resp.StatusCode)"
            }
            catch {
                # 端口在监听但请求失败 —— 多数是还在启动中
                $healthOk = $false
                $healthText = '端口在监听，请求失败'
            }
        }

        $result += [pscustomobject]@{
            Key        = $svc.Key
            Name       = $svc.Name
            Port       = $svc.Port
            Desc       = $svc.Desc
            Running    = [bool]$conn
            Pid        = if ($conn) { $conn.OwningProcess } else { $null }
            Health     = $healthOk
            HealthText = $healthText
        }
    }

    return $result
}

# 某个服务的日志文件路径（启动器"查看日志"用）
function Get-LeetNoteLogPath {
    param([Parameter(Mandatory)][string]$Key)
    return Join-Path $LeetNoteLogDir "$Key.out.log"
}
