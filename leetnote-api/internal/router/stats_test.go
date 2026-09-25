package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/router"
)

// insertNotesAtDays 插入若干条笔记，created_at 分别为「今天 - offsetDays」。
// offsets 里有重复值就表示同一天有多条记录。
func insertNotesAtDays(t *testing.T, pool *pgxpool.Pool, userID int64, offsets []int) {
	t.Helper()

	for i, offset := range offsets {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO notes (user_id, title, content_md, status, created_at)
			VALUES ($1, $2, '', 'published', current_date - $3::int)`,
			userID, fmt.Sprintf("测试笔记-%d", i), offset)
		if err != nil {
			t.Fatalf("插入测试笔记失败: %v", err)
		}
	}
}

func decodeOverview(t *testing.T, body []byte) dto.StatsOverviewResponse {
	t.Helper()
	var out dto.StatsOverviewResponse
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("解析统计响应失败: %v, body=%s", err, string(body))
	}
	return out
}

// TestStatsStreak 专门验证「连续打卡」的 gaps-and-islands 算法。
//
// 造的数据：今天、昨天×2、前天 —— 连续 3 天；再往前第 5 天单独一条（中间断档）。
// 期望：当前连续 3、最长连续 3、打卡天数 4。
func TestStatsStreak(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	// 拿到当前用户 ID（注册时返回过，这里从 /users/me 再取一次更直观）
	w := doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader(token))
	var me dto.UserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatalf("解析当前用户失败: %v", err)
	}

	insertNotesAtDays(t, pool, me.ID, []int{0, 1, 1, 2, 5})

	w = doJSON(t, r, http.MethodGet, "/api/v1/stats/overview", nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("统计接口失败: %d %s", w.Code, w.Body.String())
	}
	got := decodeOverview(t, w.Body.Bytes())

	if got.TotalNotes != 5 {
		t.Errorf("总笔记数应为 5, 实际 %d", got.TotalNotes)
	}
	if got.ActiveDays != 4 {
		t.Errorf("打卡天数应为 4（今天/昨天/前天/五天前）, 实际 %d", got.ActiveDays)
	}
	if got.CurrentStreak != 3 {
		t.Errorf("当前连续应为 3（断档那条不能算进来）, 实际 %d", got.CurrentStreak)
	}
	if got.LongestStreak != 3 {
		t.Errorf("最长连续应为 3, 实际 %d", got.LongestStreak)
	}
}

// 昨天有记录、今天没有时，连续不应被判定为中断。
func TestStatsStreakCountsYesterday(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	w := doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader(token))
	var me dto.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &me)

	insertNotesAtDays(t, pool, me.ID, []int{1, 2})

	w = doJSON(t, r, http.MethodGet, "/api/v1/stats/overview", nil, authHeader(token))
	if got := decodeOverview(t, w.Body.Bytes()); got.CurrentStreak != 2 {
		t.Errorf("昨天+前天有记录时连续应为 2, 实际 %d", got.CurrentStreak)
	}
}

// 最近一次记录在 3 天前时，连续应归零。
func TestStatsStreakBroken(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	w := doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader(token))
	var me dto.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &me)

	insertNotesAtDays(t, pool, me.ID, []int{3, 4})

	w = doJSON(t, r, http.MethodGet, "/api/v1/stats/overview", nil, authHeader(token))
	got := decodeOverview(t, w.Body.Bytes())

	if got.CurrentStreak != 0 {
		t.Errorf("最后记录在 3 天前，当前连续应为 0, 实际 %d", got.CurrentStreak)
	}
	if got.LongestStreak != 2 {
		t.Errorf("最长连续应为 2, 实际 %d", got.LongestStreak)
	}
}

// 趋势接口要补齐没有记录的日期，否则前端折线图会出现断点。
func TestStatsTrendFillsGaps(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	w := doJSON(t, r, http.MethodGet, "/api/v1/users/me", nil, authHeader(token))
	var me dto.UserResponse
	_ = json.Unmarshal(w.Body.Bytes(), &me)

	insertNotesAtDays(t, pool, me.ID, []int{0, 0, 3})

	w = doJSON(t, r, http.MethodGet, "/api/v1/stats/trend?days=7", nil, authHeader(token))
	if w.Code != http.StatusOK {
		t.Fatalf("趋势接口失败: %d %s", w.Code, w.Body.String())
	}

	var trend dto.TrendResponse
	if err := json.Unmarshal(w.Body.Bytes(), &trend); err != nil {
		t.Fatalf("解析趋势响应失败: %v", err)
	}

	if len(trend.Points) != 7 {
		t.Fatalf("应返回 7 个点（含无记录的日期）, 实际 %d", len(trend.Points))
	}

	// 最后一个点应是今天，值为 2
	today := trend.Points[len(trend.Points)-1]
	if today.Count != 2 {
		t.Errorf("今天应有 2 条, 实际 %d", today.Count)
	}

	// 校验中间确实补了 0
	zeroDays := 0
	for _, p := range trend.Points {
		if p.Count == 0 {
			zeroDays++
		}
	}
	if zeroDays != 5 { // 7 天 - 有记录的 2 天（今天 2 条、3 天前 1 条）
		t.Errorf("应有 5 天为 0, 实际 %d", zeroDays)
	}
}

// TestSearch 覆盖搜索的核心行为。
func TestSearch(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	create := func(title, content string) {
		w := doJSON(t, r, http.MethodPost, "/api/v1/notes", map[string]any{
			"title": title, "content_md": content, "status": "published",
		}, authHeader(token))
		if w.Code != http.StatusCreated {
			t.Fatalf("创建笔记失败: %d %s", w.Code, w.Body.String())
		}
	}

	create("滑动窗口练习", "关键是不含重复字符的动态窗口")
	create("二分边界", "统一用左闭右闭区间")
	create("动态窗口专题总结", "正文不含那个词")

	run := func(query string) dto.SearchResponse {
		w := doJSON(t, r, http.MethodGet, "/api/v1/search?q="+query, nil, authHeader(token))
		if w.Code != http.StatusOK {
			t.Fatalf("搜索失败: %d %s", w.Code, w.Body.String())
		}
		var out dto.SearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("解析搜索响应失败: %v", err)
		}
		return out
	}

	t.Run("正文命中", func(t *testing.T) {
		got := run("%E5%8A%A8%E6%80%81%E7%AA%97%E5%8F%A3") // 动态窗口
		if len(got.Notes) != 2 {
			t.Fatalf("应命中 2 条（一条标题、一条正文）, 实际 %d", len(got.Notes))
		}
	})

	t.Run("标题命中优先", func(t *testing.T) {
		got := run("%E5%8A%A8%E6%80%81%E7%AA%97%E5%8F%A3")
		if len(got.Notes) < 2 {
			t.Skip("命中数量不足，跳过排序断言")
		}
		if got.Notes[0].Title != "动态窗口专题总结" {
			t.Errorf("标题命中的应排在首位, 实际首位是 %q", got.Notes[0].Title)
		}
	})
}

// LIKE 通配符必须被转义，否则用户搜 "%" 会命中全部笔记。
func TestSearchEscapesLikeWildcards(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)
	token, _ := registerAndLogin(t, r, pool)

	create := func(title string) {
		w := doJSON(t, r, http.MethodPost, "/api/v1/notes", map[string]any{
			"title": title, "content_md": "", "status": "published",
		}, authHeader(token))
		if w.Code != http.StatusCreated {
			t.Fatalf("创建失败: %d", w.Code)
		}
	}

	create("进度 100% 时")
	create("普通笔记一")
	create("普通笔记二")

	t.Run("百分号是字面量", func(t *testing.T) {
		w := doJSON(t, r, http.MethodGet, "/api/v1/search?q=100%25", nil, authHeader(token))
		var got dto.SearchResponse
		_ = json.Unmarshal(w.Body.Bytes(), &got)

		if len(got.Notes) != 1 {
			t.Errorf("搜「100%%」应只命中 1 条；命中 %d 条说明 %% 被当成了通配符", len(got.Notes))
		}
	})

	t.Run("下划线是字面量", func(t *testing.T) {
		w := doJSON(t, r, http.MethodGet, "/api/v1/search?q=_", nil, authHeader(token))
		var got dto.SearchResponse
		_ = json.Unmarshal(w.Body.Bytes(), &got)

		if len(got.Notes) != 0 {
			t.Errorf("搜「_」应命中 0 条；命中 %d 条说明 _ 被当成了单字符通配符", len(got.Notes))
		}
	})
}

// 搜索同样必须受 user_id 隔离。
func TestSearchIsolatedByUser(t *testing.T) {
	pool := testPool(t)
	r := router.New(testConfig(), pool)

	tokenA, _ := registerAndLogin(t, r, pool)
	tokenB, _ := registerAndLogin(t, r, pool)

	w := doJSON(t, r, http.MethodPost, "/api/v1/notes", map[string]any{
		"title": "A 的专属笔记", "content_md": "独一无二的关键词ZZZ", "status": "published",
	}, authHeader(tokenA))
	if w.Code != http.StatusCreated {
		t.Fatalf("A 创建失败: %d", w.Code)
	}

	// B 搜同一个关键词，不应看到 A 的笔记
	w = doJSON(t, r, http.MethodGet, "/api/v1/search?q=ZZZ", nil, authHeader(tokenB))
	var got dto.SearchResponse
	_ = json.Unmarshal(w.Body.Bytes(), &got)

	if len(got.Notes) != 0 {
		t.Errorf("B 不应搜到 A 的笔记, 实际命中 %d 条", len(got.Notes))
	}

	// A 自己能搜到
	w = doJSON(t, r, http.MethodGet, "/api/v1/search?q=ZZZ", nil, authHeader(tokenA))
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got.Notes) != 1 {
		t.Errorf("A 应能搜到自己的笔记, 实际命中 %d 条", len(got.Notes))
	}
}

func TestStatsAndSearchValidation(t *testing.T) {
	r := newTestRouter(t)
	token, _ := registerAndLogin(t, r, testPool(t))

	cases := []struct {
		path string
	}{
		{"/api/v1/search"},               // 缺 q
		{"/api/v1/search?q="},            // q 为空
		{"/api/v1/search?q=x&type=bad"},  // 非法 type
		{"/api/v1/stats/trend?days=0"},   // days 越界
		{"/api/v1/stats/trend?days=999"}, // days 越界
	}

	for _, tc := range cases {
		w := doJSON(t, r, http.MethodGet, tc.path, nil, authHeader(token))
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s 应返回 422, 实际 %d", tc.path, w.Code)
		}
	}
}

// 统计与搜索都必须登录后才能访问。
func TestStatsAndSearchRequireAuth(t *testing.T) {
	r := newTestRouter(t)

	for _, path := range []string{"/api/v1/stats/overview", "/api/v1/stats/trend", "/api/v1/search?q=x"} {
		w := doJSON(t, r, http.MethodGet, path, nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s 未登录应返回 401, 实际 %d", path, w.Code)
		}
	}
}
