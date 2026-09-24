-- =============================================================
-- 000004 · GitHub OAuth 集成
--   1. password_hash 允许为 NULL（OAuth 用户没有密码）
--   2. 新增 oauth_accounts 表
-- =============================================================

-- ---------------------------------------------------------------
-- 1. 密码可空
--
-- 通过 GitHub 注册的用户没有密码。这里用 NULL 而不是
-- 「存一个随机哈希」，是为了能明确区分两种状态：
--   - password_hash IS NULL  → 该账号没有密码，登录时不校验密码
--   - password_hash IS NOT NULL → 正常校验
-- 否则无法判断「修改密码」时该不该要求输入旧密码。
-- ---------------------------------------------------------------
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- ---------------------------------------------------------------
-- 2. 第三方账号关联表
--
-- 为什么用独立表，而不是在 users 上加一个 github_id 字段：
--   a) 一个用户可能关联多个 provider（GitHub / Google / 微信…）
--   b) provider 的信息变更不会牵动 users 表结构
--   c) UNIQUE(provider, provider_uid) 天然保证
--      「同一个 GitHub 账号不会被绑到两个用户」
--
-- 【不存 access_token】：登录流程只在换取用户信息时用一次，
-- 之后不再需要。存下来等于凭空多一份可被盗用的凭证。
-- 将来若要做「同步仓库列表」，再单独加密存储。
-- ---------------------------------------------------------------
CREATE TABLE oauth_accounts (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       VARCHAR(20)  NOT NULL,   -- github
    provider_uid   VARCHAR(100) NOT NULL,   -- provider 侧的用户唯一 ID
    provider_login VARCHAR(100),            -- provider 侧的登录名（展示用）
    avatar_url     TEXT,                    -- provider 侧头像
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_oauth_provider_uid UNIQUE (provider, provider_uid)
);

CREATE INDEX idx_oauth_accounts_user ON oauth_accounts (user_id);

CREATE TRIGGER trg_oauth_accounts_updated_at
    BEFORE UPDATE ON oauth_accounts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
