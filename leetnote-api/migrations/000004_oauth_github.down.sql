-- =============================================================
-- 000004 · 回滚
-- =============================================================

DROP TRIGGER IF EXISTS trg_oauth_accounts_updated_at ON oauth_accounts;
DROP TABLE IF EXISTS oauth_accounts;

-- 注意：若库里还存在 password_hash IS NULL 的纯 OAuth 用户，
-- 下面的 ALTER 会失败——这是【刻意】的。
-- 迁移不应该悄悄删掉用户数据，请人工决定怎么处理，例如：
--   UPDATE users SET password_hash = '!unusable' WHERE password_hash IS NULL;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
