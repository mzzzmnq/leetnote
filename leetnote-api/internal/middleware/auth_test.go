package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/jwt"
)

func authTestManager() *jwt.Manager {
	return jwt.NewManager("unit-test-secret-0123456789abcdef", 15*time.Minute, time.Hour)
}

// runAuth 用给定中间件跑一次请求，返回状态码。
func runAuth(mw gin.HandlerFunc, header string) int {
	r := gin.New()
	r.Use(mw)
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestAuth_NoHeader(t *testing.T) {
	if got := runAuth(middleware.Auth(authTestManager(), nil), ""); got != http.StatusUnauthorized {
		t.Fatalf("缺少 Authorization 应 401，实际 %d", got)
	}
}

func TestAuth_BadScheme(t *testing.T) {
	if got := runAuth(middleware.Auth(authTestManager(), nil), "Token abc"); got != http.StatusUnauthorized {
		t.Fatalf("非 Bearer 方案应 401，实际 %d", got)
	}
}

func TestAuth_LongTokenWithNilResolver(t *testing.T) {
	// 没接 PAT 解析器时，ln_ 令牌应被明确拒绝，而不是当成 JWT 去解析
	if got := runAuth(middleware.Auth(authTestManager(), nil), "Bearer ln_abcdef"); got != http.StatusUnauthorized {
		t.Fatalf("未启用 PAT 时长期令牌应 401，实际 %d", got)
	}
}

func TestAuth_AcceptsJWT(t *testing.T) {
	m := authTestManager()
	pair, err := m.GeneratePair(7)
	if err != nil {
		t.Fatalf("签发 JWT 失败: %v", err)
	}
	if got := runAuth(middleware.Auth(m, nil), "Bearer "+pair.AccessToken); got != http.StatusOK {
		t.Fatalf("有效 JWT 应 200，实际 %d", got)
	}
}
