-- =============================================================
-- 000002_pgvector · 向量检索（阶段三 AI 服务用）
-- =============================================================
--
-- 【为什么这里不写 CREATE EXTENSION】
--
-- pgvector 的 control 文件里没有 `trusted = true`，只有【超级用户】
-- 才能安装它。而我们的迁移是用应用账号 leetnote 跑的，所以
--   CREATE EXTENSION IF NOT EXISTS vector;
-- 会直接报 `permission denied to create extension "vector"`，
-- 整个迁移卡在版本 2 变成 dirty。
--
-- 对比：pg_trgm / pgcrypto 从 PostgreSQL 13 起标记为 trusted，
-- 普通用户也能装，所以放在 000001 里没问题。
--
-- 正确做法是把「装扩展」和「建表」分成两件事：
--   * 装扩展 = 一次性的 DBA 前置步骤（超级用户执行）
--   * 建表   = 版本化迁移（应用账号执行）
--
-- 扩展的安装方式见 docs/ENVIRONMENT.md「向量扩展」一节。
-- =============================================================

-- 先检查，缺了就给出能直接照做的提示。
-- 不加这一步的话，后面建表会报 "type vector does not exist"，很难定位到根因。
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
        RAISE EXCEPTION
            '缺少 pgvector 扩展。请先用超级用户执行一次：'
            'psql -U postgres -h localhost -d leetnote -c "CREATE EXTENSION vector;"'
            '（详见 docs/ENVIRONMENT.md）';
    END IF;
END
$$;

-- 笔记的向量表示（维度按实际 embedding 模型调整，1536 对应 text-embedding-3-small）
CREATE TABLE note_embeddings (
    note_id    BIGINT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    embedding  vector(1536) NOT NULL,
    model      VARCHAR(64)  NOT NULL,   -- 记录用了哪个 embedding 模型
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- HNSW 索引：近似最近邻，查询快、召回高
CREATE INDEX idx_note_embeddings_hnsw
    ON note_embeddings USING hnsw (embedding vector_cosine_ops);

-- 相似题检索示例：
--   SELECT n.id, n.title, 1 - (e.embedding <=> $1) AS similarity
--   FROM note_embeddings e
--   JOIN notes n ON n.id = e.note_id
--   ORDER BY e.embedding <=> $1
--   LIMIT 10;
