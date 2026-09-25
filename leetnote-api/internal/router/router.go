package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/handler"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/oauth"
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
	oauthRepo := repository.NewOAuthRepository(pool)
	problemRepo := repository.NewProblemRepository(pool)
	tagRepo := repository.NewTagRepository(pool)
	noteRepo := repository.NewNoteRepository(pool)
	solutionRepo := repository.NewSolutionRepository(pool)
	statsRepo := repository.NewStatsRepository(pool)
	searchRepo := repository.NewSearchRepository(pool)

	githubClient := oauth.NewGitHubClient(
		cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubRedirectURL)

	authSvc := service.NewAuthService(userRepo, tokenManager)
	userSvc := service.NewUserService(userRepo)
	oauthSvc := service.NewOAuthService(userRepo, oauthRepo, githubClient, tokenManager)
	problemSvc := service.NewProblemService(problemRepo)
	tagSvc := service.NewTagService(tagRepo)
	noteSvc := service.NewNoteService(pool, noteRepo, solutionRepo, tagRepo, problemRepo)
	statsSvc := service.NewStatsService(statsRepo)
	searchSvc := service.NewSearchService(searchRepo, tagRepo, problemRepo)

	authHandler := handler.NewAuthHandler(authSvc, cfg)
	userHandler := handler.NewUserHandler(userSvc)
	oauthHandler := handler.NewOAuthHandler(oauthSvc, cfg)
	problemHandler := handler.NewProblemHandler(problemSvc)
	tagHandler := handler.NewTagHandler(tagSvc)
	noteHandler := handler.NewNoteHandler(noteSvc)
	statsHandler := handler.NewStatsHandler(statsSvc)
	searchHandler := handler.NewSearchHandler(searchSvc)
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

		// GitHub OAuth
		//   authorize: 前端拿授权地址
		//   callback : GitHub 授权后浏览器跳转过来的地址（返回 302，不是 JSON）
		auth.GET("/github/authorize", oauthHandler.Authorize)
		auth.GET("/github/callback", oauthHandler.Callback)
	}

	// 需要登录的接口
	authed := v1.Group("", middleware.Auth(tokenManager))
	{
		users := authed.Group("/users")
		{
			users.GET("/me", userHandler.Me)
			users.PATCH("/me", userHandler.UpdateMe)
			users.POST("/me/password", userHandler.ChangePassword)

			// 第三方账号绑定管理
			users.GET("/me/oauth-accounts", oauthHandler.LinkedAccounts)
			users.DELETE("/me/oauth-accounts/github", oauthHandler.Unlink)
		}

		// 题目（全局共享的元数据）
		problems := authed.Group("/problems")
		{
			problems.GET("", problemHandler.List)
			problems.POST("", problemHandler.Create)
			problems.GET("/:id", problemHandler.Get)
			problems.PUT("/:id", problemHandler.Update)
			problems.DELETE("/:id", problemHandler.Delete)
		}

		// 标签（全局共享；数量少，不分页）
		tags := authed.Group("/tags")
		{
			tags.GET("", tagHandler.List)
			tags.POST("", tagHandler.Create)
			tags.DELETE("/:id", tagHandler.Delete)
		}

		// 笔记（用户私有数据，所有操作都限定在当前用户名下）
		notes := authed.Group("/notes")
		{
			notes.GET("", noteHandler.List)
			notes.POST("", noteHandler.Create)
			notes.GET("/:id", noteHandler.Get)
			notes.PUT("/:id", noteHandler.Update)
			notes.DELETE("/:id", noteHandler.Delete)
			notes.POST("/:id/star", noteHandler.ToggleStar)

			notes.GET("/:id/solutions", noteHandler.ListSolutions)
			notes.POST("/:id/solutions", noteHandler.CreateSolution)
		}

		// 解法的独立编辑（不用为了改一个解法提交整篇笔记）
		solutions := authed.Group("/solutions")
		{
			solutions.PUT("/:id", noteHandler.UpdateSolution)
			solutions.DELETE("/:id", noteHandler.DeleteSolution)
		}

		// 统计看板
		stats := authed.Group("/stats")
		{
			stats.GET("/overview", statsHandler.Overview)
			stats.GET("/trend", statsHandler.Trend)
		}

		// 全文检索（一次返回笔记 + 题目两类结果）
		authed.GET("/search", searchHandler.Search)
	}

	// 未匹配路由统一走错误响应格式
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, errs.ErrRouteNotFound)
	})

	return r
}
