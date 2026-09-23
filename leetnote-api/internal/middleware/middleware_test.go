package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

func TestRequestIDGenerated(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, middleware.GetRequestID(c))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	body := w.Body.String()
	if body == "" {
		t.Fatal("未生成 request id")
	}
	if got := w.Header().Get(middleware.HeaderRequestID); got != body {
		t.Errorf("响应头与 context 中的 request id 不一致: %q vs %q", got, body)
	}
	if len(body) != 16 { // 8 字节 hex
		t.Errorf("request id 长度异常: %q", body)
	}
}

// 上游网关传了 X-Request-ID 时必须复用，否则全链路追踪就断了。
func TestRequestIDReusedFromHeader(t *testing.T) {
	const upstreamID = "trace-from-nginx-123"

	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/x", func(c *gin.Context) {
		c.String(http.StatusOK, middleware.GetRequestID(c))
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(middleware.HeaderRequestID, upstreamID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != upstreamID {
		t.Errorf("未复用上游 request id: %s", w.Body.String())
	}
}

func newCORSRouter(origins []string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(origins))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORSAllowedOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("白名单内的 Origin 应被允许，实际: %q", got)
	}
}

// 安全底线：不在白名单里的 Origin 绝不能回显，否则等于对所有站点开放。
func TestCORSDisallowedOrigin(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("恶意 Origin 不应被允许，实际回显: %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	r := newCORSRouter([]string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("预检请求应返回 204，实际 %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("预检响应缺少 Access-Control-Allow-Methods")
	}
}

// Recovery 的核心价值：单个请求 panic 不能拖垮整个进程。
func TestRecoveryCatchesPanic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.Recovery())
	r.GET("/boom", func(c *gin.Context) {
		panic("模拟空指针")
	})

	w := httptest.NewRecorder()
	// 这里不应该把 panic 抛出来
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic 应被转成 500，实际 %d", w.Code)
	}
}
