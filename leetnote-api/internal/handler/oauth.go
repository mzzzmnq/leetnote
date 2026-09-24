package handler

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
)

const (
	oauthStateCookie = "oauth_state"
	oauthStateTTL    = 10 * time.Minute
)

type OAuthHandler struct {
	svc *service.OAuthService
	cfg *config.Config
}

func NewOAuthHandler(svc *service.OAuthService, cfg *config.Config) *OAuthHandler {
	return &OAuthHandler{svc: svc, cfg: cfg}
}

// Authorize 处理 GET /api/v1/auth/github/authorize
//
// 前端调用它拿到授权地址，然后 window.location.href 跳过去。
func (h *OAuthHandler) Authorize(c *gin.Context) {
	if !h.svc.GitHubEnabled() {
		response.Fail(c, errs.ErrBadRequest.WithMessage("本站未启用 GitHub 登录"))
		return
	}

	state, err := h.svc.GenerateState()
	if err != nil {
		response.Fail(c, errs.ErrInternal.Wrap(err))
		return
	}

	// state 存进 httpOnly Cookie，回调时比对，用来防 CSRF
	h.setStateCookie(c, state)

	response.OK(c, dto.GitHubAuthorizeResponse{
		AuthorizeURL: h.svc.GitHubAuthorizeURL(state),
	})
}

// Callback 处理 GET /api/v1/auth/github/callback
//
// 这是 GitHub 授权后浏览器【跳转】过来的地址，不是 XHR 调用，
// 所以失败时也要重定向回前端（带上错误码），而不是返回 JSON。
func (h *OAuthHandler) Callback(c *gin.Context) {
	redirectPath := h.redirectSuffix(c.Query("redirect"))

	// ---------- 1. 校验 state（防 CSRF）----------
	if !h.verifyState(c) {
		h.redirectError(c, redirectPath, "state_invalid")
		return
	}

	// ---------- 2. 用户在 GitHub 上点了「拒绝」----------
	if c.Query("error") != "" {
		h.redirectError(c, redirectPath, "github_denied")
		return
	}

	code := c.Query("code")
	if code == "" {
		h.redirectError(c, redirectPath, "code_missing")
		return
	}

	// ---------- 3. 换 token 并拉用户信息 ----------
	gu, err := h.svc.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		h.redirectError(c, redirectPath, "exchange_failed")
		return
	}

	// ---------- 4. 登录 / 注册 / 关联 ----------
	result, err := h.svc.LoginWithGitHub(c.Request.Context(), gu)
	if err != nil {
		h.redirectError(c, redirectPath, "login_failed")
		return
	}

	// ---------- 5. 下发 refresh Cookie 并跳回前端 ----------
	//
	// 【关键】绝不把 token 放在跳转 URL 里。
	// URL 会进浏览器历史、可能被 Referer 头带出去、也会留在服务器访问日志里。
	// 正确做法：这里只下发 httpOnly 的 refresh Cookie，
	// 前端回调页再调 POST /auth/refresh 换取 access_token。
	setRefreshCookie(c, h.cfg, result.Tokens)

	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/auth/callback"+redirectPath+"status=ok")
}

// LinkedAccounts 处理 GET /api/v1/users/me/oauth-accounts
func (h *OAuthHandler) LinkedAccounts(c *gin.Context) {
	accounts, err := h.svc.ListLinkedAccounts(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}

	out := make([]dto.OAuthAccountResponse, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, dto.NewOAuthAccountResponse(a))
	}
	response.OK(c, out)
}

// Unlink 处理 DELETE /api/v1/users/me/oauth-accounts/github
func (h *OAuthHandler) Unlink(c *gin.Context) {
	if err := h.svc.UnlinkGitHub(c.Request.Context(), middleware.MustUserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// ---------------------------------------------------------------

func (h *OAuthHandler) setStateCookie(c *gin.Context, state string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   int(oauthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
}

// verifyState 比对回调参数与 Cookie 里的 state。
//
// 用 subtle.ConstantTimeCompare 而不是 == ：普通字符串比较会在
// 第一个不同字符处提前返回，理论上可以被逐字节爆破出正确值。
func (h *OAuthHandler) verifyState(c *gin.Context) bool {
	expected, err := c.Cookie(oauthStateCookie)
	if err != nil || expected == "" {
		return false
	}

	// 一次性使用：无论成功失败都立刻清除，防止重放
	h.clearStateCookie(c)

	actual := c.Query("state")
	if actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func (h *OAuthHandler) clearStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
}

// redirectSuffix 把前端传来的 redirect 参数处理成安全的查询串后缀。
//
// 只接受站内相对路径，防止开放重定向（open redirect）：
// 攻击者构造 ?redirect=https://evil.com，用户登录后就被带到钓鱼站，
// 而地址栏显示的是可信域名刚跳转过去的，极具迷惑性。
func (h *OAuthHandler) redirectSuffix(raw string) string {
	if raw == "" {
		return "?"
	}
	// 必须以单个 / 开头：排除 //evil.com（协议相对 URL）和 http://...
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/\\") {
		return "?"
	}
	return "?redirect=" + url.QueryEscape(raw) + "&"
}

func (h *OAuthHandler) redirectError(c *gin.Context, suffix, code string) {
	c.Redirect(http.StatusFound,
		h.cfg.FrontendURL+"/auth/callback"+suffix+"status=error&error="+url.QueryEscape(code))
}
