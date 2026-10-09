package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/apitoken"
)

// importBody 构造一个「外部工具导入笔记」的请求体。
func importBody(slug, title string, lcID int) map[string]any {
	return map[string]any{
		"problem": map[string]any{
			"title_slug":  slug,
			"title":       title,
			"leetcode_id": lcID,
			"difficulty":  "Medium",
			"url":         "https://leetcode.cn/problems/" + slug + "/",
		},
		"content_md": "## 思路\n双指针。",
		"solutions": []map[string]any{
			{"language": "python3", "code": "def solve():\n    pass\n"},
		},
		"source": "integration-test",
	}
}

// TestImportNote_NewThenReuse 覆盖「先新建题目、再复用同一题目」两条路径。
//
// 【为什么自带 fixture 而不依赖题单里的题】集成测试跑在开发库上，
// 若依赖「3sum 已存在」，换一台空库跑就会失败。自己造、自己清才稳。
func TestImportNote_NewThenReuse(t *testing.T) {
	pool := testPool(t)
	r := newRouterWith(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	slug := uniqueName("itest-import")
	title := "导入测试题"

	// 题目是全局表，测试必须自己清干净（否则每跑一次就多一道题）
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx,
			`DELETE FROM notes WHERE problem_id = (SELECT id FROM problems WHERE title_slug = $1)`, slug)
		_, _ = pool.Exec(ctx, `DELETE FROM problems WHERE title_slug = $1`, slug)
	})

	// ---------- 第一次：题目不存在 → 新建 ----------
	w := doJSON(t, r, http.MethodPost, "/api/v1/import/note",
		importBody(slug, title, 9999), authHeader(token))
	if w.Code != http.StatusCreated {
		t.Fatalf("导入应返回 201，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var first dto.ImportNoteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if !first.ProblemCreated {
		t.Error("首次导入应新建题目，problem_created 应为 true")
	}
	if !first.NoteCreated {
		t.Error("首次导入应新建笔记，note_created 应为 true")
	}
	if first.Note.ProblemID == nil {
		t.Error("导入的笔记应关联到题目")
	}
	if first.Note.Title != "9999. 导入测试题" {
		t.Errorf("标题应带题号，实际 %q", first.Note.Title)
	}
	if len(first.Note.Solutions) != 1 || first.Note.Solutions[0].Language != "python" {
		t.Errorf("解法语言应被归一化为 python，实际 %+v", first.Note.Solutions)
	}

	// ---------- 第二次：题目已存在、笔记也已存在 → 更新（幂等，不 409） ----------
	w = doJSON(t, r, http.MethodPost, "/api/v1/import/note",
		importBody(slug, title, 9999), authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("重复导入应更新已有笔记并返回 200，实际 %d，body=%s", w.Code, w.Body.String())
	}

	var second dto.ImportNoteResponse
	_ = json.Unmarshal(w.Body.Bytes(), &second)
	if second.ProblemCreated {
		t.Error("第二次导入应复用已有题目，problem_created 应为 false")
	}
	if second.NoteCreated {
		t.Error("第二次导入应更新已有笔记，note_created 应为 false")
	}
	if first.Note.ID != second.Note.ID {
		t.Errorf("重复导入应更新同一篇笔记，实际 %d → %d", first.Note.ID, second.Note.ID)
	}
	if first.Note.ProblemID != nil && second.Note.ProblemID != nil &&
		*first.Note.ProblemID != *second.Note.ProblemID {
		t.Error("两次导入应关联到同一个题目")
	}
}

// TestImportNote_PATAuth 覆盖长期令牌鉴权：有效令牌放行、伪造令牌 401。
func TestImportNote_PATAuth(t *testing.T) {
	pool := testPool(t)
	r := newRouterWith(testConfig(), pool)
	_, username := registerAndLogin(t, r, pool)

	var userID int64
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE username = $1`, username).Scan(&userID); err != nil {
		t.Fatalf("查用户失败: %v", err)
	}

	raw, hash, err := apitoken.Generate()
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}
	// api_tokens 随用户级联删除，registerAndLogin 的 cleanup 会一并清掉
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO api_tokens (user_id, name, token_hash) VALUES ($1, $2, $3)`,
		userID, "integration-test", hash); err != nil {
		t.Fatalf("写令牌失败: %v", err)
	}

	// 有效长期令牌 → 200
	w := doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader(raw))
	if w.Code != http.StatusOK {
		t.Fatalf("长期令牌应通过鉴权，实际 %d，body=%s", w.Code, w.Body.String())
	}

	// 伪造的 ln_ 令牌 → 401
	w = doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader("ln_deadbeefdeadbeef"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无效令牌应返回 401，实际 %d", w.Code)
	}
}
