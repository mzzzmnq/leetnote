package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/router"
)

// doJSON 发一个 JSON 请求。
func doJSON(t *testing.T, r *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func findCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func cookieHeader(w *httptest.ResponseRecorder, name string) string {
	if c := findCookie(w, name); c != nil {
		return c.Name + "=" + c.Value
	}
	return ""
}

// uniqueIdentity 生成每次运行都不同的用户名，避免测试之间互相污染。
func uniqueIdentity() (username, email string) {
	suffix := time.Now().UnixNano()
	username = fmt.Sprintf("itest_%d", suffix)
	return username, username + "@example.com"
}

func decodeAuth(t *testing.T, w *httptest.ResponseRecorder) dto.AuthResponse {
	t.Helper()
	var out dto.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return out
}

// TestAuthFlow 覆盖注册 → 鉴权访问 → 刷新 → 登出 → 冲突 → 登录失败的完整链路。
func TestAuthFlow(t *testing.T) {
	pool := testPool(t)
	cfg := &config.Config{
		AppEnv:      "test",
		CORSOrigins: []string{"http://localhost:5173"},
		JWTSecret:   "integration-test-secret-do-not-use-in-production",
		AccessTTL:   15 * time.Minute,
		RefreshTTL:  24 * time.Hour,
	}
	r := router.New(cfg, pool)

	username, email := uniqueIdentity()
	const password = "integration-test-pw-123"

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM users WHERE lower(username) = lower($1)`, username)
	})

	// ---------- 1. 注册 ----------
	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("注册应返回 201, 实际 %d, body=%s", w.Code, w.Body.String())
	}

	auth := decodeAuth(t, w)
	if auth.AccessToken == "" {
		t.Fatal("注册响应应包含 access_token")
	}
	if auth.TokenType != "Bearer" {
		t.Errorf("token_type 应为 Bearer, 实际 %s", auth.TokenType)
	}
	if auth.ExpiresIn <= 0 {
		t.Errorf("expires_in 应为正数, 实际 %d", auth.ExpiresIn)
	}
	if auth.User.Username != username {
		t.Errorf("返回的用户名不符: %s", auth.User.Username)
	}

	refreshCookie := cookieHeader(w, "refresh_token")
	if refreshCookie == "" {
		t.Fatal("应下发 refresh_token Cookie")
	}

	// Cookie 的安全属性必须齐全，少一个都会显著削弱防护
	rc := findCookie(w, "refresh_token")
	if !rc.HttpOnly {
		t.Error("refresh_token Cookie 必须是 HttpOnly（否则 XSS 能偷走）")
	}
	if rc.SameSite != http.SameSiteLaxMode {
		t.Errorf("refresh_token Cookie 应为 SameSite=Lax（防 CSRF）, 实际 %v", rc.SameSite)
	}
	if rc.Path != "/api/v1/auth" {
		t.Errorf("Cookie Path 应限定在 /api/v1/auth, 实际 %s", rc.Path)
	}
	if rc.MaxAge <= 0 {
		t.Errorf("Cookie 应有正的 MaxAge, 实际 %d", rc.MaxAge)
	}

	// refresh_token 必须只存在于 httpOnly Cookie 里，不能出现在响应体中
	if strings.Contains(w.Body.String(), "refresh_token") {
		t.Error("refresh_token 不应出现在响应体里")
	}

	// ---------- 2. 未带 Token 访问受保护接口 ----------
	w = doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录访问应返回 401, 实际 %d", w.Code)
	}

	// ---------- 3. 带 Token 访问 ----------
	w = doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, map[string]string{
		"Authorization": "Bearer " + auth.AccessToken,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("带 Token 访问应返回 200, 实际 %d, body=%s", w.Code, w.Body.String())
	}

	// 安全底线：响应里绝不能出现密码哈希
	body := strings.ToLower(w.Body.String())
	for _, leak := range []string{"password_hash", "$2a$", "$2b$", password} {
		if strings.Contains(body, strings.ToLower(leak)) {
			t.Errorf("响应泄漏了敏感信息: %s", leak)
		}
	}

	// ---------- 4. 伪造 Token 被拒绝 ----------
	w = doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, map[string]string{
		"Authorization": "Bearer " + auth.AccessToken + "x",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("被篡改的 Token 应返回 401, 实际 %d", w.Code)
	}

	// 格式不对的 Authorization 头
	w = doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, map[string]string{
		"Authorization": auth.AccessToken, // 少了 "Bearer "
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("缺少 Bearer 前缀应返回 401, 实际 %d", w.Code)
	}

	// ---------- 5. 用 refresh Cookie 换新 Token ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", nil, map[string]string{
		"Cookie": refreshCookie,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("刷新应返回 200, 实际 %d, body=%s", w.Code, w.Body.String())
	}
	if decodeAuth(t, w).AccessToken == "" {
		t.Error("刷新后应返回新的 access_token")
	}

	// ---------- 6. 修改资料 ----------
	bio := "刷题中"
	w = doJSON(t, r, http.MethodPatch, "/api/v1/users/me", map[string]*string{
		"bio": &bio,
	}, map[string]string{"Authorization": "Bearer " + auth.AccessToken})
	if w.Code != http.StatusOK {
		t.Fatalf("修改资料应返回 200, 实际 %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "刷题中") {
		t.Error("bio 未更新")
	}

	// ---------- 7. 登出清除 Cookie ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", nil, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("登出应返回 204, 实际 %d", w.Code)
	}

	lc := findCookie(w, "refresh_token")
	if lc == nil {
		t.Fatal("登出应下发清除 Cookie 的响应头")
	}
	if lc.Value != "" {
		t.Errorf("登出后 Cookie 值应为空, 实际 %q", lc.Value)
	}
	if lc.MaxAge >= 0 {
		t.Errorf("登出后 Cookie MaxAge 应为负数（立即失效）, 实际 %d", lc.MaxAge)
	}
	if !lc.HttpOnly {
		t.Error("清除 Cookie 时也要保留 HttpOnly，否则浏览器可能匹配不上原 Cookie 而不清除")
	}

	// ---------- 8. 重复注册 → 409 ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"username": username,
		"email":    "another-" + email,
		"password": password,
	}, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("重复用户名应返回 409, 实际 %d, body=%s", w.Code, w.Body.String())
	}

	// ---------- 9. 错误密码 → 401 ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login":    username,
		"password": "definitely-wrong",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码应返回 401, 实际 %d", w.Code)
	}

	// ---------- 10. 正确密码登录成功 ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"login":    username,
		"password": password,
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("正确密码登录应返回 200, 实际 %d, body=%s", w.Code, w.Body.String())
	}
}

// 参数校验失败必须返回 422，并且 details 里带上字段级提示。
func TestRegisterValidation(t *testing.T) {
	r := newTestRouter(t)

	cases := []struct {
		name  string
		body  map[string]string
		field string
	}{
		{
			name:  "密码太短",
			body:  map[string]string{"username": "validuser", "email": "a@b.com", "password": "short"},
			field: "password",
		},
		{
			name:  "邮箱格式错误",
			body:  map[string]string{"username": "validuser", "email": "not-an-email", "password": "longenough123"},
			field: "email",
		},
		{
			name:  "用户名含非法字符",
			body:  map[string]string{"username": "有中文", "email": "a@b.com", "password": "longenough123"},
			field: "username",
		},
		{
			name:  "用户名太短",
			body:  map[string]string{"username": "ab", "email": "a@b.com", "password": "longenough123"},
			field: "username",
		},
		{
			name:  "缺少必填字段",
			body:  map[string]string{"username": "validuser"},
			field: "email",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", tc.body, nil)

			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("应返回 422, 实际 %d, body=%s", w.Code, w.Body.String())
			}

			var body struct {
				Error struct {
					Code    string            `json:"code"`
					Details map[string]string `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}
			if body.Error.Code != "VALIDATION_FAILED" {
				t.Errorf("错误码应为 VALIDATION_FAILED, 实际 %s", body.Error.Code)
			}
			if _, ok := body.Error.Details[tc.field]; !ok {
				t.Errorf("details 应包含字段 %q, 实际 %v", tc.field, body.Error.Details)
			}
		})
	}
}

// 请求体不是合法 JSON 时，应返回 400 而不是 422。
func TestRegisterMalformedJSON(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("非法 JSON 应返回 400, 实际 %d, body=%s", w.Code, w.Body.String())
	}
}
