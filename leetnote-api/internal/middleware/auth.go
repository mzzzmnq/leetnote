package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

const ctxUserID = "user_id"

const bearerPrefix = "Bearer "

// Auth 校验 Authorization 头里的 access token，并把用户 ID 注入 context。
//
// 用法：
//
//	protected := v1.Group("", middleware.Auth(tokenManager))
//	protected.GET("/users/me", userHandler.Me)
func Auth(tokens *jwt.Manager) gin.HandlerFunc {
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

		// 这里显式要求 TypeAccess：refresh token 只能用来换新 token，不能直接访问业务接口
		claims, err := tokens.Parse(strings.TrimSpace(raw[len(bearerPrefix):]), jwt.TypeAccess)
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
