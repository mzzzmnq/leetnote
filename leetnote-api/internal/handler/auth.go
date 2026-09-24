package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

const (
	refreshCookieName = "refresh_token"
	// 只在认证相关的路径下携带，减少 Cookie 的暴露面
	refreshCookiePath = "/api/v1/auth"
)

type AuthHandler struct {
	svc *service.AuthService
	cfg *config.Config
}

func NewAuthHandler(svc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{svc: svc, cfg: cfg}
}

// Register 处理 POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var in dto.RegisterInput
	if !validator.BindJSON(c, &in) {
		return
	}

	result, err := h.svc.Register(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}

	setRefreshCookie(c, h.cfg, result.Tokens)
	response.Created(c, h.buildAuthResponse(result))
}

// Login 处理 POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var in dto.LoginInput
	if !validator.BindJSON(c, &in) {
		return
	}

	result, err := h.svc.Login(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}

	setRefreshCookie(c, h.cfg, result.Tokens)
	response.OK(c, h.buildAuthResponse(result))
}

// Refresh 处理 POST /api/v1/auth/refresh
//
// 从 httpOnly Cookie 读取 refresh token（前端拿不到，也不需要拿）。
func (h *AuthHandler) Refresh(c *gin.Context) {
	token, err := c.Cookie(refreshCookieName)
	if err != nil || token == "" {
		response.Fail(c, errs.ErrUnauthorized.WithMessage("未找到刷新令牌，请重新登录"))
		return
	}

	result, err := h.svc.Refresh(c.Request.Context(), token)
	if err != nil {
		// refresh token 无效就顺手清掉，避免前端反复拿着废 token 重试
		clearRefreshCookie(c, h.cfg)
		response.Fail(c, err)
		return
	}

	setRefreshCookie(c, h.cfg, result.Tokens)
	response.OK(c, h.buildAuthResponse(result))
}

// Logout 处理 POST /api/v1/auth/logout
//
// 说明：JWT 是无状态的，服务端无法「作废」已签发的 access token，
// 所以登出做两件事：
//  1. 清除 refresh token Cookie，让攻击者无法续期
//  2. access token 依靠其 15 分钟的短有效期自然失效
//
// 要做到「立即失效」需要引入 Redis 黑名单（把 jti 存进去），
// 这是 M8 的优化项。
func (h *AuthHandler) Logout(c *gin.Context) {
	clearRefreshCookie(c, h.cfg)
	response.NoContent(c)
}

func (h *AuthHandler) buildAuthResponse(r *service.AuthResult) dto.AuthResponse {
	return dto.AuthResponse{
		AccessToken: r.Tokens.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(r.Tokens.AccessExpiresAt).Seconds()),
		User:        dto.NewUserResponse(r.User),
	}
}

// setRefreshCookie 下发 refresh token。
//
// 四个安全属性缺一不可：
//   - HttpOnly  : JS 读不到，XSS 偷不走
//   - Secure    : 生产环境只在 HTTPS 上传输
//   - SameSite  : Lax 表示跨站请求不带 Cookie，缓解 CSRF
//   - Path 限定 : 只在 /api/v1/auth 下携带，减少暴露面
func setRefreshCookie(c *gin.Context, cfg *config.Config, pair *jwt.TokenPair) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    pair.RefreshToken,
		Path:     refreshCookiePath,
		MaxAge:   int(time.Until(pair.RefreshExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshCookie(c *gin.Context, cfg *config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		MaxAge:   -1, // 负数表示立即删除
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
}
