package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/apitoken"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

const ctxUserID = "user_id"

const bearerPrefix = "Bearer "

// PATResolver 把长期令牌（PAT）解析为 user_id。
//
// 【为什么定义成窄接口】中间件只需要「换 user_id」这一个能力，
// 依赖窄接口比依赖整个 repository 更好测，也避免数据访问层渗进中间件。
type PATResolver interface {
	ResolveUserID(ctx context.Context, raw string) (int64, error)
}

// Auth 校验 Authorization 头里的凭据，并把用户 ID 注入 context。
//
// 支持两种凭据：
//   - JWT（交互式登录签发，15 分钟）
//   - 长期令牌 PAT（ln_ 前缀，供本机工具 / 浏览器脚本）
//
// 两者格式不同，靠 apitoken.Prefix 分流，不会互相误判。
//
// 用法：
//
//	protected := v1.Group("", middleware.Auth(tokenManager, apiTokenRepo))
//	protected.GET("/users/me", userHandler.Me)
func Auth(tokens *jwt.Manager, pat PATResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			response.Fail(c, errs.ErrUnauthorized.WithMessage("缺少 Authorization 请求头"))
			return
		}

		if !strings.HasPrefix(raw, bearerPrefix) {
			response.Fail(c, errs.ErrUnauthorized.WithMessage("Authorization 格式应为：Bearer <token>"))
			return
		}

		credential := strings.TrimSpace(raw[len(bearerPrefix):])

		// ---- 长期令牌 ----
		if apitoken.Looks(credential) {
			if pat == nil {
				response.Fail(c, errs.ErrUnauthorized.WithMessage("本服务未启用长期令牌"))
				return
			}
			uid, err := pat.ResolveUserID(c.Request.Context(), credential)
			if err != nil {
				response.Fail(c, errs.ErrUnauthorized.Wrap(err))
				return
			}
			c.Set(ctxUserID, uid)
			c.Next()
			return
		}

		// ---- JWT ----
		// 这里显式要求 TypeAccess：refresh token 只能用来换新 token，不能直接访问业务接口
		claims, err := tokens.Parse(credential, jwt.TypeAccess)
		if err != nil {
			response.Fail(c, errs.ErrUnauthorized.Wrap(err))
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Next()
	}
}

// GetUserID 取出当前登录用户 ID。
//
// 返回值里的 bool 用来提醒调用方「这个 handler 是否挂了 Auth 中间件」。
func GetUserID(c *gin.Context) (int64, bool) {
	v, exists := c.Get(ctxUserID)
	if !exists {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// MustUserID 用于确定挂了 Auth 中间件的 handler，省去错误处理。
func MustUserID(c *gin.Context) int64 {
	id, _ := GetUserID(c)
	return id
}
