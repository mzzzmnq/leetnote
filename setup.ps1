# =============================================================
# 新机器初始化（原生方式，不用 Docker）
# =============================================================
#
# 把项目 clone 到一台新机器后，跑这一个脚本就能齐活：
#
#     .\setup.ps1
#
# 它会依次：
#     ① 检查前置工具（Go / Node / Python / PostgreSQL）
#     ② 生成三个 .env 文件（含随机密钥）
#     ③ 装前端依赖（npm install）
#     ④ 建 Python 虚拟环境并装依赖
#     ⑤ 建库 + 装扩展 + 跑迁移
#     ⑥ 编译 Go 服务
#
# 每一步都是**幂等**的：已经做过的会跳过，可以反复跑。
#
# 数据搬迁另说 —— 建好库之后用：
#     .\migrate-data.ps1 import -File <你的 dump>
#
# ---------------------------------------------------------------
# 【为什么用官方 Python 而不是 Anaconda】
# Anaconda 那套配的 OpenSSL 不兼容，会导致 pip 的所有 HTTPS 请求失败，
# 表现为「pip 找不到任何包」。这个坑很隐蔽，详见 docs/ENVIRONMENT.md。
# =============================================================

param(
    # 跳过前端依赖安装（已经装过时省时间）
    [switch]$SkipWeb,

    # 跳过 Python 虚拟环境
    [switch]$SkipAi,

    # 只检查环境，不做任何改动
    [switch]$CheckOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Root = $PSScriptRoot
. "$Root\scripts\services.ps1"
. "$Root\scripts\postgres.ps1"

# ---------------------------------------------------------------
# 输出辅助
# ---------------------------------------------------------------
$script:StepNo = 0
function Write-Head([string]$Text) {
    $script:StepNo++
    Write-Host ''
    Write-Host ('─' * 62) -ForegroundColor DarkGray
    Write-Host "  $($script:StepNo). $Text" -ForegroundColor Cyan
    Write-Host ('─' * 62) -ForegroundColor DarkGray
}
function Write-Ok([string]$Text) { Write-Host "  ✅ $Text" -ForegroundColor Green }
function Write-Skip([string]$Text) { Write-Host "  ⏭  $Text" -ForegroundColor DarkGray }
function Write-Warn2([string]$Text) { Write-Host "  ⚠️  $Text" -ForegroundColor Yellow }

# 生成随机密钥（用密码学安全的随机源，不是 Get-Random）
function New-Secret {
    param([int]$Bytes = 32)
    $buf = New-Object byte[] $Bytes
    [System.Security.Cryptography.RandomNumberGenerator]::Fill($buf)
    return ($buf | ForEach-Object { $_.ToString('x2') }) -join ''
}

Write-Host ''
Write-Host '  ╭──────────────────────────────────────────────────────────╮' -ForegroundColor DarkCyan
Write-Host '  │  LeetNote · 新机器初始化                                  │' -ForegroundColor DarkCyan
Write-Host '  ╰──────────────────────────────────────────────────────────╯' -ForegroundColor DarkCyan

# ---------------------------------------------------------------
# 1. 检查前置工具
# ---------------------------------------------------------------
Write-Head '检查前置工具'

$t = Get-LeetNoteToolchainStatus

Write-Host "     Go          : $(if ($t.GoExe) { $t.GoExe } else { '❌ 未找到' })"
Write-Host "     PostgreSQL  : $(if ($t.PgBin) { $t.PgBin } else { '❌ 未找到' })"
Write-Host "     数据目录    : $(if ($t.PgData) { $t.PgData } else { '❌ 未找到' })"

$node = Get-Command node -ErrorAction SilentlyContinue
Write-Host "     Node.js     : $(if ($node) { $node.Source } else { '❌ 未找到' })"

# Python：优先找官方安装，避开 Anaconda
$python = $null
foreach ($guess in @(
        (Join-Path $env:LOCALAPPDATA 'Programs\Python\Python311\python.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Python\Python312\python.exe')
    )) {
    if (Test-Path $guess) { $python = $guess; break }
}
if (-not $python) {
    $cmd = Get-Command python -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source -notmatch 'conda|anaconda') { $python = $cmd.Source }
}
Write-Host "     Python      : $(if ($python) { $python } else { '❌ 未找到官方 Python' })"

Write-Host ''
if (-not $t.HasGo -or -not $t.HasPg -or -not $node -or (-not $python -and -not $SkipAi)) {
    Write-Warn2 '有工具没找到。可以继续，但相关步骤会失败。'
    Write-Host '        装在非标准位置时，复制 local.config.example.ps1 为 local.config.ps1' -ForegroundColor DarkGray
    Write-Host '        指定 $LeetNoteGoExe / $LeetNotePgBin / $LeetNotePgData 即可。' -ForegroundColor DarkGray
    Write-Host ''
}

if ($CheckOnly) {
    Write-Host '  （-CheckOnly：只检查，不修改任何东西）' -ForegroundColor DarkGray
    Write-Host ''
    exit 0
}

# ---------------------------------------------------------------
# 2. 生成 .env
# ---------------------------------------------------------------
Write-Head '生成环境变量文件'

$jwtSecret = New-Secret 32
$internalToken = New-Secret 16

function New-EnvFile {
    param([string]$Path, [string]$Content, [string]$Label)

    if (Test-Path $Path) {
        Write-Skip "$Label 已存在，跳过（要重建请先删掉它）"
        return
    }
    # 注意要写成 UTF-8 无 BOM：带 BOM 时 Godotenv / pydantic 读第一行会带上不可见字符
    [System.IO.File]::WriteAllText($Path, $Content, (New-Object System.Text.UTF8Encoding($false)))
    Write-Ok "$Label 已生成"
}

New-EnvFile -Label 'leetnote-api\.env' -Path (Join-Path $Root 'leetnote-api\.env') -Content @"
# 由 setup.ps1 生成
APP_ENV=development
HTTP_PORT=8080
GRPC_PORT=9091

DATABASE_URL=postgres://$($env:LEETNOTE_DB_USER):$($env:LEETNOTE_DB_PASS)@$($env:LEETNOTE_DB_HOST):$($env:LEETNOTE_DB_PORT)/$($env:LEETNOTE_DB_NAME)?sslmode=disable
REDIS_ADDR=localhost:6379

# 每次生成都是随机的
JWT_SECRET=$jwtSecret
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h

CORS_ORIGINS=http://localhost:5173
AI_SERVICE_URL=http://127.0.0.1:8000
AI_INTERNAL_TOKEN=$internalToken

FRONTEND_URL=http://localhost:5173

# GitHub 登录（留空即关闭）
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/github/callback
"@

New-EnvFile -Label 'leetnote-ai\.env' -Path (Join-Path $Root 'leetnote-ai\.env') -Content @"
# 由 setup.ps1 生成
APP_ENV=development
HTTP_PORT=8000

# 注意 psycopg 用 postgresql:// 前缀
DATABASE_URL=postgresql://$($env:LEETNOTE_DB_USER):$($env:LEETNOTE_DB_PASS)@$($env:LEETNOTE_DB_HOST):$($env:LEETNOTE_DB_PORT)/$($env:LEETNOTE_DB_NAME)

INTERNAL_API_TOKEN=$internalToken

# Embedding：留空则用内置的本地哈希向量器（离线可用）
EMBEDDING_MODEL=
EMBEDDING_DIM=1536
LLM_API_KEY=
LLM_BASE_URL=https://api.siliconflow.cn/v1

# 对话模型（AI 讲解）。换机器要重新填自己的 key
CHAT_API_KEY=
CHAT_BASE_URL=
CHAT_MODEL=
CHAT_USER_AGENT=leetnote-ai/0.1.0
CHAT_SESSION_HEADER=
"@

New-EnvFile -Label 'frontend\.env' -Path (Join-Path $Root 'frontend\.env') -Content @"
# 由 setup.ps1 生成
# 开发时前端直连后端（不用 Vite 代理，以便真实验证 CORS 配置）
VITE_API_BASE_URL=http://localhost:8080/api/v1
"@

# ---------------------------------------------------------------
# 3. 前端依赖
# ---------------------------------------------------------------
Write-Head '安装前端依赖'

if ($SkipWeb) {
    Write-Skip '已指定 -SkipWeb'
}
elseif (Test-Path (Join-Path $Root 'frontend\node_modules')) {
    Write-Skip 'node_modules 已存在（要重装请先删掉它）'
}
else {
    Push-Location (Join-Path $Root 'frontend')
    try {
        & npm install
        if ($LASTEXITCODE -ne 0) { throw 'npm install 失败' }
        Write-Ok '前端依赖已安装'
    }
    finally { Pop-Location }
}

# ---------------------------------------------------------------
# 4. Python 虚拟环境
# ---------------------------------------------------------------
Write-Head '配置 AI 服务虚拟环境'

$venvDir = Join-Path $Root 'leetnote-ai\.venv'
if ($SkipAi) {
    Write-Skip '已指定 -SkipAi'
}
elseif (Test-Path (Join-Path $venvDir 'Scripts\python.exe')) {
    Write-Skip '.venv 已存在（要重建请先删掉它）'
}
elseif (-not $python) {
    Write-Warn2 '找不到官方 Python，跳过。请手工建虚拟环境（见 docs/DEPLOY.md）'
}
else {
    Write-Host "     用 $python 创建 .venv"
    # 【只能用官方 Python】Anaconda 的 OpenSSL 不兼容，会让所有 HTTPS 请求失败
    & $python -m venv $venvDir
    if ($LASTEXITCODE -ne 0) { throw '创建 venv 失败' }

    $venvPy = Join-Path $venvDir 'Scripts\python.exe'
    & $venvPy -m pip install -q --upgrade pip
    & $venvPy -m pip install -q -i https://mirrors.aliyun.com/pypi/simple/ `
        -r (Join-Path $Root 'leetnote-ai\requirements.txt')
    if ($LASTEXITCODE -ne 0) { throw 'pip install 失败' }
    Write-Ok '虚拟环境已就绪'
}

# ---------------------------------------------------------------
# 5. 数据库
# ---------------------------------------------------------------
Write-Head '准备数据库'

if (-not $t.HasPg) {
    Write-Warn2 '找不到 PostgreSQL，跳过建库。装好后重新跑本脚本即可。'
}
else {
    if (-not (Start-LeetNotePostgres)) { throw 'PostgreSQL 启动失败' }
    Write-Ok 'PostgreSQL 已启动'

    $psql = Get-LeetNotePsql

    # 建业务账号与数据库（已存在则跳过）
    Set-LeetNotePgPassword -SuperUser
    $roleExists = (& $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -At -c "SELECT 1 FROM pg_roles WHERE rolname = '$LeetNoteDbUser'" 2>&1 | Out-String).Trim()

    if ($roleExists -ne '1') {
        & $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -c "CREATE ROLE $LeetNoteDbUser LOGIN PASSWORD '$LeetNoteDbPass'" | Out-Null
        Write-Ok "业务账号 $LeetNoteDbUser 已创建"
    }
    else {
        Write-Skip "业务账号 $LeetNoteDbUser 已存在"
    }

    $dbExists = (& $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -At -c "SELECT 1 FROM pg_database WHERE datname = '$LeetNoteDbName'" 2>&1 | Out-String).Trim()

    if ($dbExists -ne '1') {
        & $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort -d postgres `
            -c "CREATE DATABASE $LeetNoteDbName OWNER $LeetNoteDbUser" | Out-Null
        Write-Ok "数据库 $LeetNoteDbName 已创建"
    }
    else {
        Write-Skip "数据库 $LeetNoteDbName 已存在"
    }

    Push-Location (Join-Path $Root 'leetnote-api')
    try {
        # 装扩展需要超级用户
        & $psql -U $LeetNoteDbSuperUser -h $LeetNoteDbHost -p $LeetNoteDbPort `
            -d $LeetNoteDbName -q -f 'scripts\init-extensions.sql' 2>&1 | Out-Null
        Write-Ok '扩展已就绪'

        if ($t.HasGo) {
            & (Get-LeetNoteGoExe) run ./cmd/migrate up
            if ($LASTEXITCODE -ne 0) { throw '迁移失败' }
            Write-Ok '表结构已就绪'
        }
        else {
            Write-Warn2 '没有 Go，跳过迁移。装好 Go 后执行：cd leetnote-api; .\dev.ps1 migrate-up'
        }
    }
    finally { Pop-Location }
}

# ---------------------------------------------------------------
# 6. 编译 Go 服务
# ---------------------------------------------------------------
Write-Head '编译 Go 服务'

if (-not $t.HasGo) {
    Write-Warn2 '找不到 Go，跳过'
}
else {
    Push-Location (Join-Path $Root 'leetnote-api')
    try {
        if (-not $env:GOPROXY) { $env:GOPROXY = 'https://goproxy.cn,direct' }
        New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
        & (Get-LeetNoteGoExe) build -o bin/leetnote-api.exe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw '编译失败' }
        Write-Ok 'bin\leetnote-api.exe 已生成'
    }
    finally { Pop-Location }
}

# ---------------------------------------------------------------
# 完成
# ---------------------------------------------------------------
Write-Host ''
Write-Host ('─' * 62) -ForegroundColor DarkGray
Write-Host '  🎉 初始化完成' -ForegroundColor Green
Write-Host ('─' * 62) -ForegroundColor DarkGray
Write-Host ''
Write-Host '  接下来：' -ForegroundColor Cyan
Write-Host '     1. 如果是新库且要搬数据：' -ForegroundColor Gray
Write-Host '          .\migrate-data.ps1 import -File <你的 dump 文件>' -ForegroundColor DarkGray
Write-Host '     2. 启动全部服务：' -ForegroundColor Gray
Write-Host '          双击 LeetNote.bat   或者   .\start-all.ps1' -ForegroundColor DarkGray
Write-Host '     3. 浏览器打开 http://localhost:5173' -ForegroundColor Gray
Write-Host ''
Write-Host '  想用 AI 讲解功能的话，还要在 leetnote-ai\.env 里填 CHAT_API_KEY' -ForegroundColor DarkGray
Write-Host '  （见 docs/DEPLOY.md「可选：接上大模型」）' -ForegroundColor DarkGray
Write-Host ''
