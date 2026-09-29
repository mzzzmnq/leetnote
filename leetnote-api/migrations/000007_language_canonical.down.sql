-- =============================================================
-- 000007_language_canonical · 回滚
-- =============================================================
--
-- 只删约束。
--
-- 【为什么不解归一化】把 python 拆回 py / python3 / python2 是不可能的 ——
-- 原值在归一化时就丢了，无法还原。数据迁移本身就是单向的，
-- 硬要"回滚"只能靠备份。这里保持诚实：结构回滚，数据不回滚。

ALTER TABLE solutions DROP CONSTRAINT IF EXISTS solutions_language_check;

COMMENT ON COLUMN solutions.language IS 'python / go / java / cpp ...';
