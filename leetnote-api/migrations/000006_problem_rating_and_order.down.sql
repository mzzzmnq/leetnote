-- =============================================================
-- 000006_problem_rating_and_order · 回滚
-- =============================================================

DROP INDEX IF EXISTS idx_problems_sort_order;
DROP INDEX IF EXISTS idx_problems_rating;

ALTER TABLE tags     DROP COLUMN IF EXISTS sort_order;
ALTER TABLE problems DROP COLUMN IF EXISTS sort_order;
ALTER TABLE problems DROP COLUMN IF EXISTS rating;
