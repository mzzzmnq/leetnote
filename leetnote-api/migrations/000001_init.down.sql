-- =============================================================
-- 000001_init · 回滚
-- =============================================================

DROP TRIGGER IF EXISTS trg_notes_updated_at ON notes;
DROP TABLE IF EXISTS review_logs;
DROP TABLE IF EXISTS review_cards;
DROP TABLE IF EXISTS note_tags;
DROP TABLE IF EXISTS solutions;
DROP TABLE IF EXISTS notes;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS problems;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
