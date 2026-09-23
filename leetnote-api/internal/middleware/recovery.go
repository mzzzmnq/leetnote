package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

// Recovery 捕获 panics，记录堆栈，返回 500。
//
// 为什么必须有：默认情况下一个 panic 会终止整个进程（如果是野生的 goroutine 更是直接崩），
// 有了它，单个请求出错只会返回 500，不会拖垮整个服务。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"path", c.Request.URL.Path,
					"request_id", GetRequestID(c),
					"stack", string(debug.Stack()),
				)

				if !c.Writer.Written() {
					response.Fail(c, errs.ErrInternal)
					return
				}
				c.Abort()
			}
		}()

		c.Next()
	}
}
