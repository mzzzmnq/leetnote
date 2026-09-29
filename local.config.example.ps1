# =============================================================
# 本机配置模板
# =============================================================
#
# 用法：把本文件复制成 local.config.ps1（同目录），按需修改。
#       local.config.ps1 **不会**被提交到 git —— 它是每台机器自己的配置。
#
#     Copy-Item local.config.example.ps1 local.config.ps1
#
# ---------------------------------------------------------------
# 什么时候需要它
# ---------------------------------------------------------------
#
# 大多数情况【不需要】—— 项目会自动从 PATH 上找 go 和 pg_ctl，
# 找得到就直接用。
#
# 只有下面这些情况才要手工指定：
#   * Go 或 PostgreSQL 装在非标准位置，且没加进 PATH
#   * 数据库端口 / 账号 / 密码与默认值不同
#   * 想用另一套数据库（比如把开发库和生产库分开）
#
# 全部变量都是可选的，只写你需要改的那几行即可。
# =============================================================

# ---------------------------------------------------------------
# 工具位置
# ---------------------------------------------------------------

# Go 编译器（go.exe 的完整路径，不是 bin 目录）
# $LeetNoteGoExe = 'D:\dev\go\bin\go.exe'

# PostgreSQL 的 bin 目录（里面要有 pg_ctl.exe / psql.exe / pg_dump.exe）
# $LeetNotePgBin = 'D:\dev\pgsql\bin'

# PostgreSQL 数据目录（含 postgresql.conf 和 postgresql.pid 的那个目录）
# $LeetNotePgData = 'D:\dev\pgsql\data'

# ---------------------------------------------------------------
# 数据库连接
#
# 这几个值要和 leetnote-api/.env 里的 DATABASE_URL 保持一致，
# 否则迁移/备份脚本连的库和应用连的库会是两个。
# ---------------------------------------------------------------

# $LeetNoteDbName = 'leetnote'
# $LeetNoteDbUser = 'leetnote'
# $LeetNoteDbPass = 'leetnote'
# $LeetNoteDbHost = 'localhost'
# $LeetNoteDbPort = '5432'

# 超级用户：只有「安装扩展」和「创建数据库」会用到。
# pgvector 不是 trusted 扩展，普通用户装不了，所以必须有这个。
# $LeetNoteDbSuperUser = 'postgres'
# $LeetNoteDbSuperPass = 'postgres'
