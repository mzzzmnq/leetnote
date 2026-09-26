package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mzzzmnq/leetnote-api/internal/ai"
)

func TestFindSimilarSendsInternalToken(t *testing.T) {
	var gotToken string
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Internal-Token")
		gotPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"note_id": 1,
			"model":   "test-model",
			"items": []map[string]any{
				{"note_id": 2, "title": "相似的笔记", "similarity": 0.87},
			},
		})
	}))
	defer srv.Close()

	client := ai.NewClient(srv.URL, "super-secret-token")

	items, model, err := client.FindSimilar(context.Background(), 42, 1, 5)
	if err != nil {
		t.Fatalf("调用失败: %v", err)
	}

	// 服务间共享密钥必须带上，否则 AI 服务会拒绝
	if gotToken != "super-secret-token" {
		t.Errorf("未正确传递内部令牌, 实际 %q", gotToken)
	}
	if gotPath != "/api/v1/ai/similar" {
		t.Errorf("请求路径错误: %s", gotPath)
	}

	if model != "test-model" {
		t.Errorf("model 字段错误: %s", model)
	}
	if len(items) != 1 {
		t.Fatalf("应返回 1 条, 实际 %d", len(items))
	}
	if items[0].NoteID != 2 || items[0].Similarity != 0.87 {
		t.Errorf("解析结果错误: %+v", items[0])
	}
}

// AI 服务返回错误时，必须把状态码和响应体带出来，否则排查线上问题只能抓瞎。
func TestErrorIncludesUpstreamDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail":"内部调用令牌无效"}`))
	}))
	defer srv.Close()

	client := ai.NewClient(srv.URL, "wrong")

	_, _, err := client.FindSimilar(context.Background(), 1, 1, 5)
	if err == nil {
		t.Fatal("上游返回 401 时应报错")
	}

	msg := err.Error()
	if !strings.Contains(msg, "401") || !strings.Contains(msg, "内部调用令牌无效") {
		t.Errorf("错误信息应包含状态码与上游详情, 实际: %s", msg)
	}
}

func TestDisabledClientReturnsError(t *testing.T) {
	client := ai.NewClient("", "")

	if client.Enabled() {
		t.Fatal("空地址的客户端应被视为未启用")
	}

	if _, _, err := client.FindSimilar(context.Background(), 1, 1, 5); err == nil {
		t.Error("未启用的客户端调用应返回错误，而不是静默成功")
	}
}

func TestEmbedNotePostsNoteID(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"note_id":7,"model":"m","dim":1536,"ok":true}`))
	}))
	defer srv.Close()

	client := ai.NewClient(srv.URL, "t")
	if err := client.EmbedNote(context.Background(), 7); err != nil {
		t.Fatalf("调用失败: %v", err)
	}

	if got, _ := gotBody["note_id"].(float64); int64(got) != 7 {
		t.Errorf("请求体里的 note_id 错误: %v", gotBody)
	}
}
