# =============================================================
# 本机路径配置（自动探测 + 可覆盖）
# =============================================================
#
# 【为什么需要这个文件】
#
# 原先 start-all.ps1 / stop-all.ps1 / launcher.ps1 里写死了
# `D:\dev\go`、`D:\dev\pgsql`、`D:\dev\pg-start.ps1` 这些路径。
# 换一台电脑（哪怕只是换个安装位置）就直接失效 ——
# 项目**依赖仓库外的文件**，而仓库外的文件根本不会跟着 git 过去。
#
# 现在的解析顺序：
#   1. 项目根目录的 local.config.ps1（个人覆盖，已被 git 忽略）
#   2. 自动探测（从 PATH 上找 go / pg_ctl）
#   3. 常见安装位置兜底扫描
#   4. 都找不到 → 给出明确提示，而不是莫名其妙地失败
#
# 换机器通常什么都不用配 —— 只要 Go 和 PostgreSQL 在 PATH 上。
# 装在非标准位置时，复制 local.config.example.ps1 改一下即可。
# =============================================================

Set-StrictMode -Version Latest

# 允许调用方先设好 $LocalPathsRoot（比如 services.ps1 已经算过了）。
# 单独运行时从本文件位置推。
if (-not (Get-Variable -Name LocalPathsRoot -Scope Script -ErrorAction SilentlyContinue)) {
    $LocalPathsRoot = Split-Path -Parent $PSScriptRoot
}

# ---------------------------------------------------------------
# 1. 个人覆盖优先
#
# local.config.ps1 里可以设置这几个变量（用普通变量，不用 $env:）：
#     $LeetNoteGoExe   Go 的 go.exe 完整路径
#     $LeetNotePgBin   PostgreSQL 的 bin 目录
#     $LeetNotePgData  PostgreSQL 的数据目录
#     $LeetNoteDbUser / $LeetNoteDbPass / ...   连接参数
# ---------------------------------------------------------------
$LeetNoteGoExe = $null
$LeetNotePgBin = $null
$LeetNotePgData = $null
$LeetNoteDbName = $null
$LeetNoteDbUser = $null
$LeetNoteDbPass = $null
$LeetNoteDbHost = $null
$LeetNoteDbPort = $null
$LeetNoteDbSuperUser = $null
$LeetNoteDbSuperPass = $null

$LocalConfigFile = Join-Path $LocalPathsRoot 'local.config.ps1'
if (Test-Path $LocalConfigFile) {
    . $LocalConfigFile
}

# ---------------------------------------------------------------
# 2. 探测工具
# ---------------------------------------------------------------
function Find-LeetNoteExe {
    param([Parameter(Mandatory)][string]$Name)

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { return $cmd.Source }
    return $null
}

# ---------------------------------------------------------------
# 3. Go
# ---------------------------------------------------------------
if (-not $LeetNoteGoExe) {
    $LeetNoteGoExe = Find-LeetNoteExe 'go'
}

if (-not $LeetNoteGoExe) {
    # 便携版（解压即用）通常没加进 PATH，扫一遍常见位置
    $candidates = @(
        'D:\dev\go\bin\go.exe',
        'C:\Go\bin\go.exe',
        (Join-Path $env:LOCALAPPDATA 'Programs\Go\bin\go.exe')
    )
    if ($env:ProgramFiles) { $candidates += (Join-Path $env:ProgramFiles 'Go\bin\go.exe') }

    foreach ($guess in $candidates) {
        if ($guess -and (Test-Path $guess)) { $LeetNoteGoExe = $guess; break }
    }
}

# ---------------------------------------------------------------
# 4. PostgreSQL
#
# 只需要 bin 目录 —— pg_ctl（启停）、psql（迁移）、pg_dump/pg_restore（备份还原）
# 都在同一个目录下，定位到其中一个就能推出其余。
# ---------------------------------------------------------------
if (-not $LeetNotePgBin) {
    $pgCtl = Find-LeetNoteExe 'pg_ctl'
    if ($pgCtl) { $LeetNotePgBin = Split-Path -Parent $pgCtl }
}

if (-not $LeetNotePgBin) {
    $candidates = @('D:\dev\pgsql\bin')
    foreach ($major in 18, 17, 16, 15) {
        $candidates += "C:\Program Files\PostgreSQL\$major\bin"
    }
    foreach ($guess in $candidates) {
        if (Test-Path (Join-Path $guess 'pg_ctl.exe')) { $LeetNotePgBin = $guess; break }
    }
}

# 数据目录：优先显式配置，其次 PGDATA 环境变量，
# 再其次按便携版布局猜（bin 的兄弟目录 data/）
if (-not $LeetNotePgData) {
    if ($env:PGDATA -and (Test-Path $env:PGDATA)) {
        $LeetNotePgData = $env:PGDATA
    }
    elseif ($LeetNotePgBin) {
        $sibling = Join-Path (Split-Path -Parent $LeetNotePgBin) 'data'
        if (Test-Path $sibling) { $LeetNotePgData = $sibling }
    }
}

# ---------------------------------------------------------------
# 5. 数据库连接参数
#
# 与 leetnote-api/.env 的 DATABASE_URL 是同一套值。
# 单独列出来是因为迁移/备份脚本走的是 psql 命令行参数，而不是 .env。
# ---------------------------------------------------------------
if (-not $LeetNoteDbName) { $LeetNoteDbName = 'leetnote' }
if (-not $LeetNoteDbUser) { $LeetNoteDbUser = 'leetnote' }
if (-not $LeetNoteDbPass) { $LeetNoteDbPass = 'leetnote' }
if (-not $LeetNoteDbHost) { $LeetNoteDbHost = 'localhost' }
if (-not $LeetNoteDbPort) { $LeetNoteDbPort = '5432' }

# 超级用户：只有「装扩展」和「建库」需要（pgvector 不是 trusted 扩展）
if (-not $LeetNoteDbSuperUser) { $LeetNoteDbSuperUser = 'postgres' }
if (-not $LeetNoteDbSuperPass) { $LeetNoteDbSuperPass = 'postgres' }

# 导出到 $env: 让子进程（go / psql）也能用到
$env:LEETNOTE_GO_EXE = $LeetNoteGoExe
$env:LEETNOTE_PG_BIN = $LeetNotePgBin
$env:LEETNOTE_PGDATA = $LeetNotePgData

# ---------------------------------------------------------------
# 6. 供其它脚本调用的辅助函数
# ---------------------------------------------------------------
function Get-LeetNoteGoExe {
    if (-not $LeetNoteGoExe) {
        throw '找不到 Go。请安装并加入 PATH，或在 local.config.ps1 里设置 $LeetNoteGoExe。'
    }
    return $LeetNoteGoExe
}

function Get-LeetNotePgExe {
    param([Parameter(Mandatory)][string]$Exe)

    if (-not $LeetNotePgBin) {
        throw '找不到 PostgreSQL。请安装并加入 PATH，或在 local.config.ps1 里设置 $LeetNotePgBin。'
    }
    $full = Join-Path $LeetNotePgBin $Exe
    if (-not (Test-Path $full)) {
        throw "在 $LeetNotePgBin 里找不到 $Exe"
    }
    return $full
}

<#
把密码放进 PGPASSWORD 再调 psql。
比拼 connection string 更稳 —— 密码里有 @ 或 : 时 URL 会解析错。
#>
function Set-LeetNotePgPassword {
    param([switch]$SuperUser)

    $env:PGPASSWORD = if ($SuperUser) { $LeetNoteDbSuperPass } else { $LeetNoteDbPass }
}

function Get-LeetNotePsql {
    return Get-LeetNotePgExe 'psql.exe'
}

# 探测结果的摘要（status / 启动器展示用）
function Get-LeetNoteToolchainStatus {
    return [pscustomobject]@{
        GoExe    = $LeetNoteGoExe
        PgBin    = $LeetNotePgBin
        PgData   = $LeetNotePgData
        DbName   = $LeetNoteDbName
        HasGo    = [bool]$LeetNoteGoExe
        HasPg    = [bool]$LeetNotePgBin
        HasData  = [bool]$LeetNotePgData
        ConfigFile = if (Test-Path $LocalConfigFile) { $LocalConfigFile } else { $null }
    }
}
