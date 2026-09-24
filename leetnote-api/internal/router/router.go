package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/handler"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

// New 装配依赖、中间件与路由。
//
// 依赖装配采用「手工构造函数注入」而不是引入 wire/fx 之类的 DI 框架：
// 这个体量的项目，显式装配比框架更清晰、更好调试，
// 而且依赖关系一眼可见——出了问题不用去猜框架做了什么。
//
// 中间件顺序很重要（洋葱模型，先进后出）：
//
//	RequestID → Logger → Recovery → CORS → [Auth] → handler
//
// RequestID 放最外层，后续所有日志才能带上它；
// Recovery 放在 Logger 之内，panic 转 500 时 Logger 仍能记录到状态码。
func New(cfg *config.Config, pool *pgxpool.Pool) *gin.Engine {
	// 注册自定义校验规则（username 等），必须在任何请求进来之前完成
	validator.Register()

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// ---------- 依赖装配 ----------
	tokenManager := jwt.NewManager(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)

	userRepo := repository.NewUserRepository(pool)

	authSvc := service.NewAuthService(userRepo, tokenManager)
	userSvc := service.NewUserService(userRepo)

	authHandler := handler.NewAuthHandler(authSvc, cfg)
	userHandler := handler.NewUserHandler(userSvc)
	healthHandler := handler.NewHealthHandler(pool)

	// ---------- 全局中间件 ----------
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Logger(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORSOrigins),
	)

	// ---------- 探针 ----------
	r.GET("/health", healthHandler.Check)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/live", healthHandler.Live)

	// ---------- API v1 ----------
	v1 := r.Group("/api/v1")

	// 公开接口：认证
	//
	// logout 刻意不要求登录：access token 已过期时，
	// 用户仍然应该能清掉 refresh token Cookie 完成登出。
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// 需要登录的接口
	authed := v1.Group("", middleware.Auth(tokenManager))
	{
		users := authed.Group("/users")
		{
			users.GET("/me", userHandler.Me)
			users.PATCH("/me", userHandler.UpdateMe)
			users.POST("/me/password", userHandler.ChangePassword)
		}
	}

	// 未匹配路由统一走错误响应格式
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, errs.ErrRouteNotFound)
	})

	return r
}
