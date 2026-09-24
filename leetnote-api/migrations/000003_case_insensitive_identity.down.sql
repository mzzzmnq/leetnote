-- =============================================================
-- 000003_case_insensitive_identity · 回滚
-- =============================================================

DROP INDEX IF EXISTS uq_users_username_lower;
DROP INDEX IF EXISTS uq_users_email_lower;
