package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderRequestID 是透传请求 ID 的 HTTP 头。
	HeaderRequestID = "X-Request-ID"
	// ctxRequestID 是存入 gin.Context 的键名。
	ctxRequestID = "request_id"
)

// RequestID 为每个请求生成唯一 ID。
//
// 作用：一次请求会在多个中间件、handler、SQL 日志里留下痕迹，
// 有了 request_id 才能把它们串成一条完整的链路（排查线上问题的刚需）。
// 如果上游（网关/Nginx）已经传了 X-Request-ID，就复用它，保证全链路一致。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			b := make([]byte, 8)
			if _, err := rand.Read(b); err == nil {
				id = hex.EncodeToString(b)
			} else {
				id = "unknown"
			}
		}

		c.Set(ctxRequestID, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

// GetRequestID 从 context 中取出请求 ID。
func GetRequestID(c *gin.Context) string {
	return c.GetString(ctxRequestID)
}
