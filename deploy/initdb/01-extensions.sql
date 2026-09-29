-- =============================================================
-- PostgreSQL 容器首次初始化时自动执行
-- =============================================================
--
-- 放在 /docker-entrypoint-initdb.d/ 下的 .sql 文件，会被官方镜像在
-- **数据目录为空时**自动执行一次，而且是以超级用户身份 ——
-- 正好满足「装扩展需要超级用户」这个要求。
--
-- 【注意】只在数据目录为空时跑。已经初始化过的卷不会重跑，
-- 这是对的：重复执行没有意义，而且扩展本身也带了 IF NOT EXISTS。
-- 真要重建，得先 docker compose down -v 把卷删掉。
--
-- 【为什么不能把这几行放进 migrations】
-- pgvector 的 control 文件里没有 trusted = true，普通用户装不了。
-- 迁移是用应用账号跑的，写进去必然 permission denied。
-- 详见 leetnote-api/scripts/init-extensions.sql 的说明。
-- =============================================================

-- 中文子串检索的三元组索引
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 向量类型与 HNSW 索引（非 trusted，必须超级用户）
CREATE EXTENSION IF NOT EXISTS vector;
