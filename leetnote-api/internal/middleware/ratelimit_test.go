package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/cache"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

func newRateLimitClient(t *testing.T) *cache.Client {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	t.Cleanup(mr.Close)

	c := cache.New(context.Background(), mr.Addr())
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// hit 发一次请求，返回状态码与响应头。
func hit(r *gin.Engine, method, path, remoteAddr string) (int, http.Header) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remoteAddr
	r.ServeHTTP(w, req)
	return w.Code, w.Header()
}

func TestRateLimit_ByIPWhenAnonymous(t *testing.T) {
	c := newRateLimitClient(t)
	r := gin.New()
	r.Use(middleware.RateLimit(c, middleware.RateLimitConfig{
		Enabled: true, Window: time.Minute, Max: 100, AuthMax: 2,
	}))
	r.POST("/api/v1/auth/login", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })

	addr := "203.0.113.7:5555"
	for i := 1; i <= 2; i++ {
		code, hdr := hit(r, http.MethodPost, "/api/v1/auth/login", addr)
		if code != http.StatusOK {
			t.Fatalf("第 %d 次 = %d，期望 200", i, code)
		}
		if hdr.Get("X-RateLimit-Limit") != "2" {
			t.Errorf("第 %d 次 X-RateLimit-Limit = %q，期望 2", i, hdr.Get("X-RateLimit-Limit"))
		}
	}

	code, hdr := hit(r, http.MethodPost, "/api/v1/auth/login", addr)
	if code != http.StatusTooManyRequests {
		t.Fatalf("第 3 次 = %d，期望 429", code)
	}
	if hdr.Get("Retry-After") == "" {
		t.Error("429 响应应带 Retry-After 头")
	}

	// 换一个 IP 不应该受影响
	if code, _ := hit(r, http.MethodPost, "/api/v1/auth/login", "198.51.100.9:5555"); code != http.StatusOK {
		t.Errorf("不同 IP 应独立计数，得到 %d，期望 200", code)
	}
}

func TestRateLimit_ByUserIDWhenAuthed(t *testing.T) {
	c := newRateLimitClient(t)
	r := gin.New()
	// 模拟 Auth 中间件注入 user_id
	r.Use(func(ctx *gin.Context) {
		ctx.Set("user_id", int64(42))
		ctx.Next()
	})
	r.Use(middleware.RateLimit(c, middleware.RateLimitConfig{
		Enabled: true, Window: time.Minute, Max: 2, AuthMax: 100,
	}))
	r.GET("/api/v1/notes", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })

	addr := "203.0.113.7:5555"
	for i := 1; i <= 2; i++ {
		if code, _ := hit(r, http.MethodGet, "/api/v1/notes", addr); code != http.StatusOK {
			t.Fatalf("第 %d 次 = %d，期望 200", i, code)
		}
	}
	if code, hdr := hit(r, http.MethodGet, "/api/v1/notes", addr); code != http.StatusTooManyRequests {
		t.Fatalf("第 3 次 = %d，期望 429", code)
	} else if hdr.Get("X-RateLimit-Limit") != "2" {
		t.Errorf("X-RateLimit-Limit = %q，期望 2", hdr.Get("X-RateLimit-Limit"))
	}
}

func TestRateLimit_DisabledAlwaysPasses(t *testing.T) {
	c := newRateLimitClient(t)
	r := gin.New()
	r.Use(middleware.RateLimit(c, middleware.RateLimitConfig{
		Enabled: false, Window: time.Minute, Max: 1, AuthMax: 1,
	}))
	r.GET("/x", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })

	for i := range 5 {
		if code, _ := hit(r, http.MethodGet, "/x", "203.0.113.7:1"); code != http.StatusOK {
			t.Fatalf("关闭限流时第 %d 次 = %d，期望 200", i+1, code)
		}
	}
}

func TestRateLimit_DisabledRedisFailsOpen(t *testing.T) {
	c := cache.New(context.Background(), "") // 未启用
	r := gin.New()
	r.Use(middleware.RateLimit(c, middleware.RateLimitConfig{
		Enabled: true, Window: time.Minute, Max: 1, AuthMax: 1,
	}))
	r.GET("/x", func(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{"ok": true}) })

	for i := range 5 {
		if code, _ := hit(r, http.MethodGet, "/x", "203.0.113.7:1"); code != http.StatusOK {
			t.Fatalf("Redis 不可用时第 %d 次 = %d，期望 200（fail-open）", i+1, code)
		}
	}
}
