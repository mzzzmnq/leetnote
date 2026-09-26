package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	leetcodeGraphQLEndpoint = "https://leetcode.com/graphql"
	// LeetCode 的分页上限是 100，不能再大
	lcPageSize = 100
)

// 只需要 (slug -> 难度) 这一个映射：题单里已经给了中文标题和题号。
const lcProblemsQuery = `query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    total: totalNum
    questions: data {
      questionFrontendId
      title
      titleSlug
      difficulty
    }
  }
}`

type lcQuestion struct {
	FrontendID string `json:"questionFrontendId"`
	Title      string `json:"title"`
	TitleSlug  string `json:"titleSlug"`
	Difficulty string `json:"difficulty"`
}

type lcPage struct {
	Total     int          `json:"total"`
	Questions []lcQuestion `json:"questions"`
}

type lcResponse struct {
	Data struct {
		List lcPage `json:"problemsetQuestionList"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchDifficulties 拉取全部题目，返回 slug -> 难度 的映射。
//
// 题单 markdown 里没有难度字段，只能从 LeetCode 官方接口补。
// 用 slug 做关联键（题号会变，slug 稳定）。
func FetchDifficulties(ctx context.Context) (map[string]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// 4096 是当前题库量级，预分配避免反复扩容
	out := make(map[string]string, 4096)

	total := -1
	for skip := 0; ; skip += lcPageSize {
		page, pageTotal, err := fetchPage(ctx, client, skip)
		if err != nil {
			return nil, err
		}

		if total < 0 {
			total = pageTotal
		}

		for _, q := range page {
			out[q.TitleSlug] = q.Difficulty
		}

		if len(page) == 0 || skip+lcPageSize >= total {
			break
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("未能从 LeetCode 拉到任何题目")
	}
	return out, nil
}

func fetchPage(ctx context.Context, client *http.Client, skip int) ([]lcQuestion, int, error) {
	payload, err := json.Marshal(map[string]any{
		"query": lcProblemsQuery,
		"variables": map[string]any{
			"categorySlug": "",
			"limit":        lcPageSize,
			"skip":         skip,
			"filters":      map[string]any{},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("构造请求体失败: %w", err)
	}

	// LeetCode 偶尔会 429 / 502，重试几次再放弃
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			leetcodeGraphQLEndpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, 0, fmt.Errorf("构造请求失败: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		// LeetCode 要求带这几个头，否则可能被拦
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; leetnote-importer/1.0)")
		req.Header.Set("Referer", "https://leetcode.com/problemset/")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求 LeetCode 失败: %w", err)
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()

		if readErr != nil {
			lastErr = fmt.Errorf("读取响应失败: %w", readErr)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("LeetCode 返回 %d", resp.StatusCode)
			continue
		}

		var parsed lcResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			lastErr = fmt.Errorf("解析响应失败: %w", err)
			continue
		}
		if len(parsed.Errors) > 0 {
			return nil, 0, fmt.Errorf("LeetCode 返回错误: %s", parsed.Errors[0].Message)
		}

		return parsed.Data.List.Questions, parsed.Data.List.Total, nil
	}

	return nil, 0, lastErr
}
