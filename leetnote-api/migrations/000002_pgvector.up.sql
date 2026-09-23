-- =============================================================
-- 000002_pgvector · 向量检索（阶段三 AI 服务用）
--
-- ⚠️ 需要先安装 pgvector 扩展。EDB 便携版不含，安装方式见
--    docs/ENVIRONMENT.md「pgvector 安装」一节。
--    安装前执行本迁移会报错：extension "vector" is not available
-- =============================================================

CREATE EXTENSION IF NOT EXISTS vector;

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
