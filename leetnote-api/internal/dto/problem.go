package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

// ProblemInput 用于创建与全量更新题目（PUT 语义，字段全部必填）。
//
// 为什么题目用 PUT 而不是 PATCH：题目字段少且编辑表单一定提交完整对象，
// 部分更新的复杂度换不来收益。
type ProblemInput struct {
	LeetCodeID *int    `json:"leetcode_id" binding:"omitempty,min=1,max=9999"`
	Title      string  `json:"title" binding:"required,max=200"`
	TitleSlug  string  `json:"title_slug" binding:"required,max=200"`
	Difficulty string  `json:"difficulty" binding:"required,oneof=Easy Medium Hard"`
	URL        *string `json:"url" binding:"omitempty,url,max=500"`
}

// ProblemQuery 是题目列表的筛选参数。
type ProblemQuery struct {
	Keyword    string `form:"keyword" binding:"max=100"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=Easy Medium Hard"`
	PageQuery
}

type ProblemResponse struct {
	ID         int64     `json:"id"`
	LeetCodeID *int      `json:"leetcode_id"`
	Title      string    `json:"title"`
	TitleSlug  string    `json:"title_slug"`
	Difficulty string    `json:"difficulty"`
	URL        *string   `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewProblemResponse(p *model.Problem) ProblemResponse {
	if p == nil {
		return ProblemResponse{}
	}
	return ProblemResponse{
		ID:         p.ID,
		LeetCodeID: p.LeetCodeID,
		Title:      p.Title,
		TitleSlug:  p.TitleSlug,
		Difficulty: p.Difficulty,
		URL:        p.URL,
		CreatedAt:  p.CreatedAt,
	}
}

func NewProblemResponses(list []*model.Problem) []ProblemResponse {
	out := make([]ProblemResponse, 0, len(list))
	for _, p := range list {
		out = append(out, NewProblemResponse(p))
	}
	return out
}

// ProblemBrief 是嵌套在笔记里的精简题目信息（只要展示需要的字段）。
type ProblemBrief struct {
	ID         int64  `json:"id"`
	LeetCodeID *int   `json:"leetcode_id"`
	Title      string `json:"title"`
	Difficulty string `json:"difficulty"`
}

func NewProblemBrief(p *model.Problem) *ProblemBrief {
	if p == nil {
		return nil
	}
	return &ProblemBrief{
		ID:         p.ID,
		LeetCodeID: p.LeetCodeID,
		Title:      p.Title,
		Difficulty: p.Difficulty,
	}
}
