package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/handler"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

// New 组装中间件与路由。
//
// 中间件顺序很重要（洋葱模型，先进后出）：
//
//	RequestID → Logger → Recovery → CORS → handler
//
// RequestID 放最外层，这样后续所有日志都能带上它；
// Recovery 放在 Logger 之内，panic 时 Logger 仍能记录到 500 状态码。
func New(cfg *config.Config, pool *pgxpool.Pool) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORSOrigins),
	)

	// ---------- 探针 ----------
	health := handler.NewHealthHandler(pool)
	r.GET("/health", health.Check)
	r.GET("/ready", health.Ready)
	r.GET("/live", health.Live)

	// ---------- 业务 API ----------
	v1 := r.Group("/api/v1")
	{
		// M2 起在这里挂载：
		//   auth    := v1.Group("/auth")
		//   users   := v1.Group("/users")
		//   notes   := v1.Group("/notes")
		_ = v1
	}

	// 未匹配路由统一走错误响应格式
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, errs.ErrRouteNotFound)
	})

	return r
}
