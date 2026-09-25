package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/router"
)

// uniqueName 生成每次都不同的名字，避免测试之间互相污染。
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

// registerAndLogin 注册一个用户并返回 (access token, 用户名)。
// 测试结束会自动清理该用户（其笔记、解法、标签关联走级联删除）。
func registerAndLogin(t *testing.T, r *gin.Engine, pool *pgxpool.Pool) (string, string) {
	t.Helper()

	username := uniqueName("itest")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM users WHERE lower(username) = lower($1)`, username)
	})

	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username": username,
		"email":    username + "@example.com",
		"password": "integration-pw-123",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("注册失败: HTTP %d, body=%s", w.Code, w.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析注册响应失败: %v", err)
	}
	return resp.AccessToken, username
}

func authHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// createNoteFixture 建好标签与题目，返回它们的 ID。
func createNoteFixture(t *testing.T, r *gin.Engine, token string) (tagID, problemID int64) {
	t.Helper()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	w := doJSON(t, r, http.MethodPost, "/api/v1/tags", map[string]any{
		"name": "动态规划" + suffix, "slug": "dp-" + suffix, "kind": "algorithm",
	}, authHeader(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("创建标签失败: %d %s", w.Code, w.Body.String())
	}
	var tag dto.TagResponse
	_ = json.Unmarshal(w.Body.Bytes(), &tag)

	w = doJSON(t, r, http.MethodPost, "/api/v1/problems", map[string]any{
		"leetcode_id": 1, "title": "Two Sum", "title_slug": "two-sum-" + suffix, "difficulty": "Easy",
	}, authHeader(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("创建题目失败: %d %s", w.Code, w.Body.String())
	}
	var problem dto.ProblemResponse
	_ = json.Unmarshal(w.Body.Bytes(), &problem)

	return tag.ID, problem.ID
}

func noteInput(problemID, tagID int64) map[string]any {
	return map[string]any{
		"problem_id": problemID,
		"title":      "两数之和 · 哈希表",
		"content_md": "## 思路\n用哈希表把查找降到 O(1)",
		"summary":    "空间换时间",
		"status":     "published",
		"tag_ids":    []int64{tagID},
		"solutions": []map[string]any{
			{"title": "哈希表", "language": "go", "code": "func twoSum(){}", "time_complexity": "O(n)", "space_complexity": "O(n)"},
			{"title": "暴力枚举", "language": "python", "code": "for i in ...", "time_complexity": "O(n^2)"},
		},
	}
}

func decodeNote(t *testing.T, body []byte) dto.NoteResponse {
	t.Helper()
	var n dto.NoteResponse
	if err := json.Unmarshal(body, &n); err != nil {
		t.Fatalf("解析笔记响应失败: %v, body=%s", err, string(body))
	}
	return n
}

// TestNoteLifecycle 覆盖笔记的完整生命周期。
func TestNoteLifecycle(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	tagID, problemID := createNoteFixture(t, r, token)

	// ---------- 创建 ----------
	w := doJSON(t, r, http.MethodPost, "/api/v1/notes", noteInput(problemID, tagID), authHeader(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("创建笔记失败: %d %s", w.Code, w.Body.String())
	}

	created := decodeNote(t, w.Body.Bytes())
	if created.ID == 0 {
		t.Fatal("应返回笔记 ID")
	}
	if len(created.Tags) != 1 {
		t.Errorf("标签数应为 1, 实际 %d", len(created.Tags))
	}
	if len(created.Solutions) != 2 {
		t.Errorf("解法数应为 2, 实际 %d", len(created.Solutions))
	}
	if created.Problem == nil || created.Problem.Title != "Two Sum" {
		t.Errorf("应带出关联题目, 实际 %+v", created.Problem)
	}
	if created.SolutionCount != 2 {
		t.Errorf("solution_count 应为 2, 实际 %d", created.SolutionCount)
	}

	// ---------- 详情 ----------
	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/notes/%d", created.ID), nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("读取详情失败: %d", w.Code)
	}
	if got := decodeNote(t, w.Body.Bytes()); got.ContentMD == "" {
		t.Error("详情应包含正文")
	}

	// ---------- 列表：不能带正文 ----------
	w = doJSON(t, r, http.MethodGet, "/api/v1/notes", nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("列表失败: %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "content_md") {
		t.Error("列表响应不应包含 content_md（正文可能上万字，会让响应体积膨胀）")
	}

	// ---------- 收藏切换 ----------
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/notes/%d/star", created.ID), nil, authHeader(token))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"is_starred":true`) {
		t.Fatalf("第一次收藏应变为 true: %d %s", w.Code, w.Body.String())
	}
	w = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/notes/%d/star", created.ID), nil, authHeader(token))
	if !strings.Contains(w.Body.String(), `"is_starred":false`) {
		t.Fatalf("第二次应变回 false: %s", w.Body.String())
	}

	// ---------- 全量更新 ----------
	updated := noteInput(problemID, tagID)
	updated["title"] = "两数之和（已更新）"
	updated["status"] = "draft"
	updated["solutions"] = []map[string]any{
		{"title": "只剩一个解法", "language": "go", "code": "x"},
	}

	w = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/notes/%d", created.ID), updated, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("更新失败: %d %s", w.Code, w.Body.String())
	}
	got := decodeNote(t, w.Body.Bytes())
	if got.Title != "两数之和（已更新）" || got.Status != "draft" {
		t.Errorf("更新未生效: title=%s status=%s", got.Title, got.Status)
	}
	if len(got.Solutions) != 1 {
		t.Errorf("解法应被替换为 1 个, 实际 %d", len(got.Solutions))
	}

	// ---------- 删除 ----------
	w = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/notes/%d", created.ID), nil, authHeader(token))
	if w.Code != http.StatusNoContent {
		t.Fatalf("删除应返回 204, 实际 %d", w.Code)
	}
	w = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/notes/%d", created.ID), nil, authHeader(token))
	if w.Code != http.StatusNotFound {
		t.Fatalf("删除后应 404, 实际 %d", w.Code)
	}
}

// TestNoteCrossUserIsolation 是本项目最重要的安全测试：
// 用户之间必须完全隔离，且越权访问返回 404 而不是 403
// （403 会泄漏「这个 ID 确实存在」）。
func TestNoteCrossUserIsolation(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)

	tokenA, _ := registerAndLogin(t, r, pool)
	tokenB, _ := registerAndLogin(t, r, pool)

	tagID, problemID := createNoteFixture(t, r, tokenA)

	w := doJSON(t, r, http.MethodPost, "/api/v1/notes", noteInput(problemID, tagID), authHeader(tokenA))
	if w.Code != http.StatusCreated {
		t.Fatalf("A 创建笔记失败: %d", w.Code)
	}
	noteA := decodeNote(t, w.Body.Bytes())
	solutionID := noteA.Solutions[0].ID

	notePath := fmt.Sprintf("/api/v1/notes/%d", noteA.ID)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"读别人的笔记", http.MethodGet, notePath, nil},
		{"改别人的笔记", http.MethodPut, notePath, noteInput(problemID, tagID)},
		{"删别人的笔记", http.MethodDelete, notePath, nil},
		{"收藏别人的笔记", http.MethodPost, notePath + "/star", nil},
		{"读别人的解法", http.MethodGet, notePath + "/solutions", nil},
		{"往别人笔记加解法", http.MethodPost, notePath + "/solutions", map[string]any{"title": "x", "language": "go", "code": "x"}},
		{"改别人的解法", http.MethodPut, fmt.Sprintf("/api/v1/solutions/%d", solutionID), map[string]any{"title": "x", "language": "go", "code": "x"}},
		{"删别人的解法", http.MethodDelete, fmt.Sprintf("/api/v1/solutions/%d", solutionID), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, tc.method, tc.path, tc.body, authHeader(tokenB))
			if w.Code != http.StatusNotFound {
				t.Errorf("越权访问应返回 404（而非 403，避免泄漏资源是否存在）, 实际 %d, body=%s",
					w.Code, w.Body.String())
			}
		})
	}

	// B 的列表里不应出现 A 的笔记
	w = doJSON(t, r, http.MethodGet, "/api/v1/notes", nil, authHeader(tokenB))
	var page struct {
		Total int64 `json:"total"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &page)
	if page.Total != 0 {
		t.Errorf("B 的列表应看不到 A 的笔记, 实际 total=%d", page.Total)
	}

	// A 本人仍然正常，确认不是「整体坏掉」
	w = doJSON(t, r, http.MethodGet, notePath, nil, authHeader(tokenA))
	if w.Code != http.StatusOK {
		t.Errorf("A 访问自己的笔记应成功, 实际 %d", w.Code)
	}
}

// TestNoteValidation 覆盖参数校验与非法关联。
func TestNoteValidation(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	cases := []struct {
		name     string
		body     map[string]any
		wantCode int
	}{
		{
			name:     "缺标题",
			body:     map[string]any{"content_md": "", "status": "draft"},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "非法状态",
			body:     map[string]any{"title": "x", "content_md": "", "status": "nope"},
			wantCode: http.StatusUnprocessableEntity,
		},
		{
			name:     "不存在的标签",
			body:     map[string]any{"title": "x", "content_md": "", "status": "draft", "tag_ids": []int64{999999999}},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "不存在的题目",
			body:     map[string]any{"title": "x", "content_md": "", "status": "draft", "problem_id": 999999999},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "解法缺必填字段",
			body: map[string]any{
				"title": "x", "content_md": "", "status": "draft",
				"solutions": []map[string]any{{"title": "没写语言"}},
			},
			wantCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doJSON(t, r, http.MethodPost, "/api/v1/notes", tc.body, authHeader(token))
			if w.Code != tc.wantCode {
				t.Errorf("期望 %d, 实际 %d, body=%s", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

// 未登录访问笔记接口必须被鉴权中间件拦下。
func TestNoteRequiresAuth(t *testing.T) {
	r := newTestRouter(t)

	for _, path := range []string{"/api/v1/notes", "/api/v1/notes/1", "/api/v1/problems", "/api/v1/tags"} {
		w := doJSON(t, r, http.MethodGet, path, nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s 未登录应返回 401, 实际 %d", path, w.Code)
		}
	}
}
