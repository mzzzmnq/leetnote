-- =============================================================
-- 000001_init · LeetNote 核心表结构
-- 对应 docs/DESIGN.md 第 5.2 节
-- =============================================================

-- ---------- 扩展 ----------
CREATE EXTENSION IF NOT EXISTS pg_trgm;      -- 中文子串检索（三元组索引）
CREATE EXTENSION IF NOT EXISTS pgcrypto;     -- gen_random_uuid()

-- ---------- updated_at 触发器函数 ----------
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============ 用户 ============
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url    TEXT,
    bio           VARCHAR(200),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============ 题目（全局共享的元数据） ============
CREATE TABLE problems (
    id          BIGSERIAL PRIMARY KEY,
    leetcode_id INTEGER,
    title       VARCHAR(200) NOT NULL,
    title_slug  VARCHAR(200) NOT NULL UNIQUE,   -- 如 two-sum，用于幂等导入
    difficulty  VARCHAR(10)  NOT NULL
                CHECK (difficulty IN ('Easy', 'Medium', 'Hard')),
    url         TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_problems_difficulty ON problems (difficulty);

-- ============ 标签 ============
CREATE TABLE tags (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(50) NOT NULL UNIQUE,     -- 展示名，如「动态规划」
    slug       VARCHAR(50) NOT NULL UNIQUE,     -- 机器名，如 dynamic-programming
    kind       VARCHAR(20) NOT NULL DEFAULT 'algorithm'
               CHECK (kind IN ('algorithm', 'data_structure', 'topic')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ 笔记 ============
CREATE TABLE notes (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    problem_id BIGINT          REFERENCES problems(id) ON DELETE SET NULL,
    title      VARCHAR(200) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    summary    VARCHAR(500),
    status     VARCHAR(10) NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'published')),
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notes_user_created ON notes (user_id, created_at DESC);
CREATE INDEX idx_notes_problem      ON notes (problem_id);

-- 同一用户对同一题目只允许一篇笔记（但允许不关联题目的自由笔记）
CREATE UNIQUE INDEX uq_notes_user_problem
    ON notes (user_id, problem_id) WHERE problem_id IS NOT NULL;

-- 中文子串检索索引
CREATE INDEX idx_notes_title_trgm   ON notes USING GIN (title gin_trgm_ops);
CREATE INDEX idx_notes_content_trgm ON notes USING GIN (content_md gin_trgm_ops);

CREATE TRIGGER trg_notes_updated_at
    BEFORE UPDATE ON notes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============ 解法 ============
CREATE TABLE solutions (
    id               BIGSERIAL PRIMARY KEY,
    note_id          BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    title            VARCHAR(100) NOT NULL,     -- 如「哈希表 · 一次遍历」
    language         VARCHAR(20)  NOT NULL,     -- python / go / java / cpp ...
    code             TEXT NOT NULL,
    time_complexity  VARCHAR(50),               -- O(n)
    space_complexity VARCHAR(50),               -- O(1)
    sort_order       INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_solutions_note ON solutions (note_id, sort_order);

-- ============ 笔记 ↔ 标签（多对多） ============
CREATE TABLE note_tags (
    note_id BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX idx_note_tags_tag ON note_tags (tag_id);

-- ============ 复习卡（SM-2 算法） ============
CREATE TABLE review_cards (
    id               BIGSERIAL PRIMARY KEY,
    note_id          BIGINT NOT NULL UNIQUE REFERENCES notes(id) ON DELETE CASCADE,
    ease_factor      REAL    NOT NULL DEFAULT 2.5,   -- 难度系数
    interval_days    INTEGER NOT NULL DEFAULT 0,     -- 当前间隔（天）
    repetitions      INTEGER NOT NULL DEFAULT 0,     -- 连续答对次数
    due_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_review_cards_due ON review_cards (due_at);

CREATE TABLE review_logs (
    id            BIGSERIAL PRIMARY KEY,
    card_id       BIGINT NOT NULL REFERENCES review_cards(id) ON DELETE CASCADE,
    rating        SMALLINT NOT NULL CHECK (rating BETWEEN 0 AND 5),
    prev_interval INTEGER,
    next_interval INTEGER,
    reviewed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_review_logs_card ON review_logs (card_id, reviewed_at DESC);
