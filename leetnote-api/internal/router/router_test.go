package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/router"
)

func init() { gin.SetMode(gin.TestMode) }

// testPool 连本地开发库。数据库不可用时跳过（而不是失败），
// 这样 CI 上没起 PG 也不会把整个测试套件搞红。
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://leetnote:leetnote@localhost:5432/leetnote?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("跳过集成测试：无法创建连接池 (%v)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("跳过集成测试：数据库不可用 (%v)", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

// testConfig 返回测试用的完整配置。
func testConfig() *config.Config {
	return &config.Config{
		AppEnv:      "test",
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "integration-test-secret-do-not-use-in-production",
		AccessTTL:   15 * time.Minute,
		RefreshTTL:  24 * time.Hour,
	}
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return router.New(testConfig(), testPool(t))
}

func do(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestLiveProbe(t *testing.T) {
	w := do(t, newTestRouter(t), http.MethodGet, "/live")

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: %d", w.Code)
	}
	if w.Body.String() != `{"status":"alive"}` {
		t.Errorf("响应体不符: %s", w.Body.String())
	}
}

func TestReadyProbe(t *testing.T) {
	w := do(t, newTestRouter(t), http.MethodGet, "/ready")

	if w.Code != http.StatusOK {
		t.Fatalf("数据库正常时就绪探针应为 200, 实际 %d", w.Code)
	}
}

func TestHealthCheck(t *testing.T) {
	w := do(t, newTestRouter(t), http.MethodGet, "/health")

	if w.Code != http.StatusOK {
		t.Fatalf("状态码错误: %d", w.Code)
	}

	var body struct {
		Status   string `json:"status"`
		Service  string `json:"service"`
		Database string `json:"database"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应体解析失败: %v", err)
	}

	if body.Status != "ok" {
		t.Errorf("status 应为 ok, 实际 %s", body.Status)
	}
	if body.Service != "leetnote-api" {
		t.Errorf("service 名称错误: %s", body.Service)
	}
	if body.Database != "ok" {
		t.Errorf("database 应为 ok, 实际 %s", body.Database)
	}
}

// 每个响应都应带上 X-Request-ID，方便全链路排查。
func TestRequestIDHeaderPresent(t *testing.T) {
	w := do(t, newTestRouter(t), http.MethodGet, "/live")

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("响应缺少 X-Request-ID")
	}
}

// 未匹配的路由也必须走统一错误格式，而不是 gin 默认的 404 纯文本。
func TestNoRouteUsesUnifiedError(t *testing.T) {
	w := do(t, newTestRouter(t), http.MethodGet, "/api/v1/does-not-exist")

	if w.Code != http.StatusNotFound {
		t.Fatalf("状态码错误: 期望 404, 实际 %d", w.Code)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("404 响应不是统一 JSON 格式: %s", w.Body.String())
	}
	if body.Error.Code != "ROUTE_NOT_FOUND" {
		t.Errorf("错误码错误: %s", body.Error.Code)
	}
}
