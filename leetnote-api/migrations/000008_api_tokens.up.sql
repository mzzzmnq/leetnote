-- =============================================================
-- 000008_api_tokens · 长期访问令牌（供本机工具 / 浏览器脚本调用 API）
-- =============================================================
--
-- 场景：像 openleet 这样的本机助手需要写入笔记，但 JWT 的 access token
-- 只有 15 分钟，refresh 又走 httpOnly Cookie —— 在跨站 / 脚本场景都不可靠。
-- 所以给「非交互式登录」的客户端发一把长期令牌。
--
-- 【安全要点】
--   * 只存 sha256 哈希，不存明文 —— 库被读到也拿不到可直接使用的令牌
--   * 令牌带 ln_ 前缀，Auth 中间件据此把它与 JWT 分流（两者格式不同）
--   * 可随时吊销（删行）；ON DELETE CASCADE 跟随用户删除
-- =============================================================

CREATE TABLE api_tokens (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(100) NOT NULL,              -- 用途标签，如 "openleet 插件"
    token_hash   CHAR(64)    NOT NULL UNIQUE,        -- sha256 十六进制
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_tokens_user ON api_tokens (user_id);

COMMENT ON TABLE api_tokens IS
    '长期 API 令牌（PAT）。只存 sha256 哈希；前缀 ln_ 用于与 JWT 分流。用 cmd/token 生成。';
COMMENT ON COLUMN api_tokens.token_hash IS '明文令牌的 sha256 十六进制；明文只在校验时由客户端提供。';
