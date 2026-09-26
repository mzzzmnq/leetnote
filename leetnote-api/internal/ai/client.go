// Package ai 封装对 leetnote-ai（Python）服务的调用。
//
// 为什么独立成包：这是【外部依赖】，涉及 HTTP 调用、超时、错误处理。
// 隔离出来后，service 层只需要面对干净的领域对象。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 是 leetnote-ai 的 HTTP 客户端。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			// AI 服务内部可能调远程 embedding 模型，给宽一点
			Timeout: 60 * time.Second,
		},
	}
}

// Enabled 表示是否配置了 AI 服务地址。未配置时相关功能自动降级。
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

// SimilarNote 是相似题结果里的一条。
type SimilarNote struct {
	NoteID     int64   `json:"note_id"`
	Title      string  `json:"title"`
	Summary    *string `json:"summary"`
	IsStarred  bool    `json:"is_starred"`
	Similarity float64 `json:"similarity"`
}

type similarResponse struct {
	Model string        `json:"model"`
	Items []SimilarNote `json:"items"`
}

// EmbedNote 请求为某篇笔记生成（或重建）向量。
func (c *Client) EmbedNote(ctx context.Context, noteID int64) error {
	_, err := c.post(ctx, "/api/v1/ai/embed", map[string]any{"note_id": noteID}, nil)
	return err
}

// FindSimilar 查询与指定笔记相似的笔记。
//
// userID 由调用方（Go 的 service 层）在完成鉴权后传入 ——
// AI 服务不自己做用户认证，完全信任这个字段。
func (c *Client) FindSimilar(ctx context.Context, userID, noteID int64, limit int) ([]SimilarNote, string, error) {
	var out similarResponse

	_, err := c.post(ctx, "/api/v1/ai/similar", map[string]any{
		"note_id": noteID,
		"user_id": userID,
		"limit":   limit,
	}, &out)
	if err != nil {
		return nil, "", err
	}

	return out.Items, out.Model, nil
}

func (c *Client) post(ctx context.Context, path string, payload any, out any) (int, error) {
	if !c.Enabled() {
		return 0, fmt.Errorf("AI 服务未配置")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// 服务间共享密钥：AI 服务只应由本服务调用
	req.Header.Set("X-Internal-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("调用 AI 服务失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("读取 AI 服务响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("AI 服务返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("解析 AI 服务响应失败: %w", err)
		}
	}
	return resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
