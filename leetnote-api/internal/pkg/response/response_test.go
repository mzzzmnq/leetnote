package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

func init() { gin.SetMode(gin.TestMode) }

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	return c, w
}

func decodeError(t *testing.T, body []byte) response.ErrorBody {
	t.Helper()
	var eb response.ErrorBody
	if err := json.Unmarshal(body, &eb); err != nil {
		t.Fatalf("响应体不是合法 JSON: %v, body=%s", err, string(body))
	}
	return eb
}

func TestFailTranslatesAppError(t *testing.T) {
	c, w := newCtx()

	response.Fail(c, errs.NotFound("NOTE", "笔记"))

	if w.Code != http.StatusNotFound {
		t.Errorf("状态码错误: 期望 404, 实际 %d", w.Code)
	}

	eb := decodeError(t, w.Body.Bytes())
	if eb.Error.Code != "NOTE_NOT_FOUND" {
		t.Errorf("错误码错误: %s", eb.Error.Code)
	}
	if eb.Error.Message != "笔记不存在" {
		t.Errorf("提示文案错误: %s", eb.Error.Message)
	}
}

// 安全底线：未识别的错误必须是 500，且响应体里不能出现底层错误内容，
// 否则会把数据库表结构之类的内部信息泄漏给攻击者。
func TestFailHidesInternalDetail(t *testing.T) {
	c, w := newCtx()
	secret := `pq: relation "users" does not exist`

	response.Fail(c, errors.New(secret))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("状态码错误: 期望 500, 实际 %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "users") {
		t.Errorf("响应体泄漏了内部错误信息: %s", w.Body.String())
	}

	eb := decodeError(t, w.Body.Bytes())
	if eb.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("错误码错误: %s", eb.Error.Code)
	}
}

func TestFailKeepsValidationDetails(t *testing.T) {
	c, w := newCtx()

	response.Fail(c, errs.ErrValidation.WithDetails(map[string]string{"email": "格式不正确"}))

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("状态码错误: 期望 422, 实际 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "email") {
		t.Errorf("Details 丢失: %s", w.Body.String())
	}
}

func TestOKAndCreated(t *testing.T) {
	c, w := newCtx()
	response.OK(c, gin.H{"hello": "world"})
	if w.Code != http.StatusOK {
		t.Errorf("OK 状态码错误: %d", w.Code)
	}

	c2, w2 := newCtx()
	response.Created(c2, gin.H{"id": 1})
	if w2.Code != http.StatusCreated {
		t.Errorf("Created 状态码错误: %d", w2.Code)
	}
}

func TestNoContent(t *testing.T) {
	c, w := newCtx()
	response.NoContent(c)
	if w.Code != http.StatusNoContent {
		t.Errorf("NoContent 状态码错误: %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("204 不应有响应体，实际: %s", w.Body.String())
	}
}

func TestPageComputesPages(t *testing.T) {
	cases := []struct {
		total     int64
		size      int
		wantPages int
	}{
		{42, 20, 3}, // 42/20 → 向上取整 3
		{40, 20, 2}, // 整除
		{0, 20, 0},  // 空
		{1, 20, 1},  // 不足一页
	}

	for _, tc := range cases {
		c, w := newCtx()
		response.Page(c, []string{"a"}, tc.total, 1, tc.size)

		var pd response.PageData[string]
		if err := json.Unmarshal(w.Body.Bytes(), &pd); err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if pd.Pages != tc.wantPages {
			t.Errorf("total=%d size=%d: 期望 pages=%d, 实际 %d", tc.total, tc.size, tc.wantPages, pd.Pages)
		}
		if pd.Total != tc.total {
			t.Errorf("total 字段错误: %d", pd.Total)
		}
	}
}
