package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 处理跨域请求。
//
// 为什么需要：前后端分离时，前端跑在 http://localhost:5173，
// 后端在 http://localhost:8080，浏览器同源策略会拦截请求。
//
// 安全要点：
//   - 只允许白名单里的 Origin，绝不能简单粗暴地反射任意 Origin
//   - 允许携带凭证时必须写具体 Origin，不能用 "*"
//   - maxAge 让浏览器缓存预检结果，减少 OPTIONS 请求
func CORS(allowOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowOrigins))
	for _, o := range allowOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if _, ok := allowed[origin]; ok && origin != "" {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			h.Set("Access-Control-Expose-Headers", "X-Request-ID")
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Max-Age", "86400")
			h.Add("Vary", "Origin")
		}

		// 预检请求直接返回 204，不进入业务 handler
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
