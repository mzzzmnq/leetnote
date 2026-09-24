-- =============================================================
-- 000003_case_insensitive_identity · 用户名 / 邮箱改为大小写不敏感唯一
--
-- 背景：
--   VARCHAR 上的 UNIQUE 是【大小写敏感】的，于是 "Alice" 和 "alice"
--   可以同时注册成功。但用户登录时根本分不清自己注册的是哪个，
--   既造成困惑，也让「用户名唯一」这个约束形同虚设。
--
-- 方案：用【表达式索引】对 lower(column) 建唯一索引。
--   相比 CITEXT 类型或 citext 扩展，表达式索引：
--     - 不需要安装额外扩展（云数据库常常不给装）
--     - 对现有列类型零侵入，迁移可逆
--
-- 注意：原有的 users_username_key / users_email_key 大小写敏感约束
--       依然保留（更宽松，不冲突），只是真正起作用的是下面这两个。
-- =============================================================

CREATE UNIQUE INDEX uq_users_username_lower ON users (lower(username));
CREATE UNIQUE INDEX uq_users_email_lower    ON users (lower(email));
