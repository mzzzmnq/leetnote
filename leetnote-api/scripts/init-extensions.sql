-- =============================================================
-- 一次性前置步骤：安装所需的 PostgreSQL 扩展
-- =============================================================
--
-- 【必须用超级用户执行】，且只需执行一次：
--
--     psql -U postgres -h localhost -d leetnote -f scripts/init-extensions.sql
--
-- 或者用 dev.ps1 里封装好的任务：
--
--     .\dev.ps1 db-init
--
-- ---------------------------------------------------------------
-- 为什么这些语句不放在 migrations/ 里
-- ---------------------------------------------------------------
--
-- pgvector 的 control 文件里没有 `trusted = true`，只有超级用户能安装。
-- 而迁移是用应用账号（leetnote）跑的，把 CREATE EXTENSION vector 写进迁移
-- 会导致迁移永远卡住并变成 dirty —— 这不是假设，是本项目真实踩过的坑：
-- 之前 6 个迁移文件里没有一个能由应用账号完整跑通，表一直是手工建的。
--
-- 职责要分清：
--   * 扩展   = DBA 一次性前置步骤（超级用户）→ 本文件
--   * 表结构 = 版本化迁移（应用账号）      → migrations/
--
-- 对比：pg_trgm / pgcrypto 从 PostgreSQL 13 起标记为 trusted，普通用户也能装，
-- 所以它们同时也写在 000001 里（IF NOT EXISTS，重复执行无害）。
-- 这里一并列出来，是为了让「全新环境初始化」只需要跑这一个文件。
-- =============================================================

-- 中文子串检索的三元组索引（trusted，应用账号也能建）
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 向量类型与 HNSW / IVFFlat 索引（【非】trusted，必须超级用户）
CREATE EXTENSION IF NOT EXISTS vector;

-- 确认结果
SELECT extname AS "已安装扩展", extversion AS "版本"
FROM pg_extension
WHERE extname IN ('pg_trgm', 'pgcrypto', 'vector')
ORDER BY extname;
