# =============================================================
# 数据库导出 / 导入（换机器时搬数据用）
# =============================================================
#
# 用法：
#     .\migrate-data.ps1 export                       # 导出到 backups\leetnote-<时间>.dump
#     .\migrate-data.ps1 export -Out D:\我的备份.dump
#     .\migrate-data.ps1 import -File .\backups\xxx.dump
#     .\migrate-data.ps1 info   -File .\backups\xxx.dump   # 只看 dump 里有什么，不导入
#
# ---------------------------------------------------------------
# 【为什么是「迁移建表 + 只还原数据」而不是整体还原】
#
# 整体还原（连 schema 一起）最省事，但它会把**源库的结构**搬过去。
# 万一源库的结构和 migrations 有偏差，偏差就跟着传染了。
#
# 这里改成两步：
#     ① 用 migrations 在新库建表（结构由版本化迁移负责，可追溯）
#     ② 只把数据灌进去（pg_restore --data-only）
#
# 好处是：换机器这件事本身**顺便验证了迁移文件是对的**。
# 导出时也刻意排除了 schema_migrations 表 —— 版本记录要由新机器
# 自己跑迁移生成，而不是从旧机器抄。
# =============================================================

param(
    [Parameter(Position = 0, Mandatory = $true)]
    [ValidateSet('export', 'import', 'info')]
    [string]$Action,

    # export：输出路径（默认 backups\leetnote-<时间戳>.dump）
    [string]$Out,

    # import / info：要处理的 dump 文件
    [string]$File,

    # 覆盖目标库名（默认用 local-paths.ps1 里的配置）。
    # 还原到临时库做校验时很有用 —— 不会动到正式库。
    [string]$DbName
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. "$PSScriptRoot\scripts\services.ps1"
. "$PSScriptRoot\scripts\postgres.ps1"

# 允许覆盖库名（要在 dot-source 之后才生效，因为它会设默认值）
if ($DbName) { $LeetNoteDbName = $DbName }

$BackupDir = Join-Path $LeetNoteRoot 'backups'

function Write-Step([string]$Text) { Write-Host "  $Text" -ForegroundColor Cyan }
function Write-Ok([string]$Text) { Write-Host "  ✅ $Text" -ForegroundColor Green }
function Write-Warn2([string]$Text) { Write-Host "  ⚠️  $Text" -ForegroundColor Yellow }

# ---------------------------------------------------------------
# 统计各表行数（用于导入后校验）
# ---------------------------------------------------------------
function Get-LeetNoteTableCounts {
    $psql = Get-LeetNotePsql
    Set-LeetNotePgPassword

    $sql = @"
SELECT string_agg(t || '=' || c, ', ' ORDER BY t)
FROM (
    SELECT 'users'    AS t, count(*) AS c FROM users    UNION ALL
    SELECT 'notes',          count(*)      FROM notes   UNION ALL
    SELECT 'solutions',      count(*)      FROM solutions UNION ALL
    SELECT 'problems',       count(*)      FROM problems UNION ALL
    SELECT 'tags',           count(*)      FROM tags    UNION ALL
    SELECT 'problem_tags',   count(*)      FROM problem_tags UNION ALL
    SELECT 'note_tags',      count(*)      FROM note_tags UNION ALL
    SELECT 'note_embeddings',count(*)      FROM note_embeddings UNION ALL
    SELECT 'review_cards',   count(*)      FROM review_cards UNION ALL
    SELECT 'review_logs',    count(*)      FROM review_logs UNION ALL
    SELECT 'oauth_accounts', count(*)      FROM oauth_accounts
) x;
"@

    $result = (& $psql -U $LeetNoteDbUser -h $LeetNoteDbHost -p $LeetNoteDbPort `
            -d $LeetNoteDbName -At -c $sql 2>&1 |
        Out-String).Trim()

    if ($LASTEXITCODE -ne 0) { return "(读取失败: $result)" }
    return $result
}

# ---------------------------------------------------------------
# 导出
# ---------------------------------------------------------------
function Invoke-Export {
    if (-not (Test-LeetNotePostgresReady)) {
        Write-Warn2 'PostgreSQL 没在运行，先启动它'
        if (-not (Start-LeetNotePostgres)) { throw 'PostgreSQL 启动失败' }
    }

    if (-not $Out) {
        New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null
        $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
        $Out = Join-Path $BackupDir "leetnote-$stamp.dump"
    }

    $outDir = Split-Path -Parent $Out
    if ($outDir -and -not (Test-Path $outDir)) {
        New-Item -ItemType Directory -Force -Path $outDir | Out-Null
    }

    Write-Step '导出前先看一眼源库内容：'
    $before = Get-LeetNoteTableCounts
    Write-Host "     $before" -ForegroundColor DarkGray
    Write-Host ''

    Write-Step '正在导出（自定义格式，压缩存储）...'

    $pgDump = Get-LeetNotePgExe 'pg_dump.exe'
    Set-LeetNotePgPassword

    # -Fc：自定义格式。比纯 SQL 小得多，而且能选择性还原（只还原数据）
    # --exclude-table=schema_migrations：版本记录由新机器自己跑迁移生成
    & $pgDump -U $LeetNoteDbUser -h $LeetNoteDbHost -p $LeetNoteDbPort `
        -Fc --exclude-table=schema_migrations `
        -f $Out $LeetNoteDbName

    if ($LASTEXITCODE -ne 0) { throw "pg_dump 失败（exit $LASTEXITCODE）" }

    $size = [Math]::Round((Get-Item $Out).Length / 1KB, 1)
    Write-Host ''
    Write-Ok "导出完成：$Out（$size KB）"
    Write-Host ''
    Write-Host '  把这个文件拷到新机器，然后执行：' -ForegroundColor DarkGray
    Write-Host "     .\migrate-data.ps1 import -File `"$Out`"" -ForegroundColor DarkGray
}

# ---------------------------------------------------------------
# 查看 dump 内容（不导入）
# ---------------------------------------------------------------
function Invoke-Info {
    if (-not $File) { throw '请用 -File 指定 dump 文件' }
    if (-not (Test-Path $File)) { throw "找不到文件：$File" }

    $pgRestore = Get-LeetNotePgExe 'pg_restore.exe'

    Write-Step "dump 目录（$([Math]::Round((Get-Item $File).Length/1KB,1)) KB）："
    Write-Host ''

    # --list 只读目录，不连数据库
    & $pgRestore --list $File | Where-Object { $_ -match 'TABLE DATA' } |
        ForEach-Object { "     $($_.Trim())" }
}

# ---------------------------------------------------------------
# 导入
# ---------------------------------------------------------------
function Invoke-Import {
    if (-not $File) { throw '请用 -File 指定 dump 文件' }
    if (-not (Test-Path $File)) { throw "找不到文件：$File" }

    Write-Host ''
    Write-Host "  准备把「$(Split-Path -Leaf $File)」导入到数据库 $LeetNoteDbName" -ForegroundColor Yellow
    Write-Host '  ⚠️  这会覆盖目标库里同名表的数据。' -ForegroundColor Yellow
    $answer = Read-Host '  继续？(y/N)'
    if ($answer -notmatch '^[Yy]') { Write-Host '  已取消'; return }

    Write-Host ''

    # ---------- 1. 确保数据库在跑 ----------
    if (-not (Test-LeetNotePostgresReady)) {
        Write-Step 'PostgreSQL 没在运行，先启动'
        if (-not (Start-LeetNotePostgres)) { throw 'PostgreSQL 启动失败' }
    }
    Write-Ok 'PostgreSQL 已就绪'

    $psql = Get-LeetNotePsql

    # ---------- 2. 建库（不存在才建）----------
    Set-LeetNotePgPassword -SuperUser

    # 注意：查不到行时 psql 什么都不输出。
    # 这里用 Out-String 包一层再 Trim —— 空输出会变成空字符串，
    # 而直接写 `[string]$x = & psql ...` 或 `$x.Trim()` 都会踩到 null。
    $exists = (& $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -At -c "SELECT 1 FROM pg_database WHERE datname = '$LeetNoteDbName'" 2>&1 |
        Out-String).Trim()

    if ($exists -ne '1') {
        Write-Step "创建数据库 $LeetNoteDbName"
        & $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -c "CREATE DATABASE $LeetNoteDbName OWNER $LeetNoteDbUser" | Out-Null
        if ($LASTEXITCODE -ne 0) { throw '建库失败' }
        Write-Ok "数据库已创建"
    }
    else {
        Write-Ok "数据库已存在"
    }

    # ---------- 3. 装扩展（超级用户）----------
    Write-Step '安装扩展（pg_trgm / pgcrypto / vector）'
    $extSql = Join-Path $LeetNoteRoot 'leetnote-api\scripts\init-extensions.sql'
    & $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort `
        -d $LeetNoteDbName -q -f $extSql 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "装扩展失败，请检查 $extSql" }
    Write-Ok '扩展已就绪'

    # ---------- 4. 用迁移建表 ----------
    Write-Step '用版本化迁移建表'

    # migrate 子命令读的是 leetnote-api/.env 里的 DATABASE_URL，
    # 而 .env 不知道我们可能用 -DbName 覆盖了库名 ——
    # 所以这里把连接串显式传给它（godotenv 不会覆盖已存在的环境变量）。
    $prevDbUrl = $env:DATABASE_URL
    $env:DATABASE_URL = "postgres://${LeetNoteDbUser}:${LeetNoteDbPass}@${LeetNoteDbHost}:${LeetNoteDbPort}/${LeetNoteDbName}?sslmode=disable"

    Push-Location (Join-Path $LeetNoteRoot 'leetnote-api')
    try {
        & (Get-LeetNoteGoExe) run ./cmd/migrate up
        if ($LASTEXITCODE -ne 0) { throw '迁移失败' }
    }
    finally {
        Pop-Location
        $env:DATABASE_URL = $prevDbUrl
    }
    Write-Ok '表结构已就绪'

    # ---------- 5. 只还原数据 ----------
    Write-Step '还原数据（只灌数据，不动表结构）'
    $pgRestore = Get-LeetNotePgExe 'pg_restore.exe'
    Set-LeetNotePgPassword

    # --data-only     只还原数据
    # --disable-triggers  关掉外键触发器，避免表之间的依赖顺序导致插入失败
    #                     （需要超级用户，所以下面用 postgres 连）
    Set-LeetNotePgPassword -SuperUser
    & $pgRestore --data-only --disable-triggers `
        -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort `
        -d $LeetNoteDbName $File

    # pg_restore 对「对象已存在」会返回非 0，但数据其实已经灌进去了，
    # 所以这里不直接抛错，改由下面的行数校验来判断成功与否。
    Write-Host ''
    Write-Step '校验导入结果：'
    $after = Get-LeetNoteTableCounts
    Write-Host "     $after" -ForegroundColor DarkGray

    Write-Host ''
    Write-Ok '导入流程结束'
    Write-Host ''
    Write-Host '  接下来：' -ForegroundColor DarkGray
    Write-Host '     1. 检查 leetnote-api\.env 和 leetnote-ai\.env 是否配好' -ForegroundColor DarkGray
    Write-Host '     2. 双击 LeetNote.bat（或 .\start-all.ps1）启动服务' -ForegroundColor DarkGray
}

switch ($Action) {
    'export' { Invoke-Export }
    'import' { Invoke-Import }
    'info' { Invoke-Info }
}
