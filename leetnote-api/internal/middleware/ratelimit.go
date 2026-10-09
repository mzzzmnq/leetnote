package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/cache"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

// RateLimitConfig 是限流参数。
type RateLimitConfig struct {
	Enabled bool // 总开关
	Window  time.Duration
	Max     int // 已登录用户：窗口内允许的请求数
	AuthMax int // 未登录（认证接口）：按 IP 计，额度更严
}

// RateLimit 基于 Redis 滑动窗口做接口限流。
//
// 计费主体：
//   - 已登录（Auth 中间件先跑过）→ user:<id>
//   - 未登录                     → ip:<client ip>
//
// 额度：已登录用 Max，未登录用 AuthMax —— 所以本中间件挂在两组上：
// 认证组（无 user_id）自然走 IP + AuthMax，受保护组走 user_id + Max。
//
// 【降级策略】Redis 不可用、或本次检查出错时一律放行（fail-open），
// 只记一条告警。限流是保护性能力，不该反过来把正常请求打断。
func RateLimit(client *cache.Client, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled || !client.Enabled() {
			c.Next()
			return
		}

		limit := cfg.AuthMax
		key := "rl:ip:" + c.ClientIP()
		if uid, ok := GetUserID(c); ok && uid > 0 {
			limit = cfg.Max
			key = fmt.Sprintf("rl:user:%d", uid)
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		allowed, remaining, err := client.AllowN(ctx, key, limit, cfg.Window)
		if err != nil {
			slog.Warn("限流检查失败，放行本次请求",
				"key", key, "request_id", GetRequestID(c), "error", err)
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			// Retry-After 是 HTTP 标准头，客户端可据此退避重试。
			c.Header("Retry-After", strconv.Itoa(int(cfg.Window.Seconds())))
			response.Abort(c, errs.ErrTooManyReqs)
			return
		}

		c.Next()
	}
}
