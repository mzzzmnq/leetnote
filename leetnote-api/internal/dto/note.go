package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

// NoteInput 用于创建与全量更新笔记（PUT 语义）。
//
// Solutions 内嵌在笔记里一起提交：编辑器的形态就是「一篇笔记 + N 个解法」，
// 分多次请求提交会让前后端都变复杂，而且无法保证原子性。
type NoteInput struct {
	ProblemID *int64  `json:"problem_id"`
	Title     string  `json:"title" binding:"required,max=200"`
	ContentMD string  `json:"content_md" binding:"max=100000"`
	Summary   *string `json:"summary" binding:"omitempty,max=500"`
	Status    string  `json:"status" binding:"required,oneof=draft published"`
	IsStarred bool    `json:"is_starred"`

	// dive 表示继续校验切片里的每个元素
	TagIDs    []int64         `json:"tag_ids" binding:"max=20"`
	Solutions []SolutionInput `json:"solutions" binding:"max=10,dive"`
}

// NoteQuery 是笔记列表的筛选参数。
type NoteQuery struct {
	Keyword    string `form:"keyword" binding:"max=100"`
	Difficulty string `form:"difficulty" binding:"omitempty,oneof=Easy Medium Hard"`
	TagID      int64  `form:"tag_id"`
	Status     string `form:"status" binding:"omitempty,oneof=draft published"`
	Starred    *bool  `form:"starred"`
	ProblemID  int64  `form:"problem_id"`
	Sort       string `form:"sort" binding:"omitempty,oneof=created updated created_asc title"`
	PageQuery
}

// NoteListItem 是列表项。
//
// 【刻意不含 content_md】：一篇笔记正文可能上万字，列表页 20 条全带上
// 会让响应体积膨胀几十倍，而列表根本用不到正文。详情接口才返回全文。
type NoteListItem struct {
	ID            int64         `json:"id"`
	ProblemID     *int64        `json:"problem_id"`
	Problem       *ProblemBrief `json:"problem"`
	Title         string        `json:"title"`
	Summary       *string       `json:"summary"`
	Status        string        `json:"status"`
	IsStarred     bool          `json:"is_starred"`
	Tags          []TagResponse `json:"tags"`
	SolutionCount int           `json:"solution_count"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// NoteResponse 是详情，包含正文、题目、标签与全部解法。
type NoteResponse struct {
	ID            int64              `json:"id"`
	ProblemID     *int64             `json:"problem_id"`
	Problem       *ProblemBrief      `json:"problem"`
	Title         string             `json:"title"`
	ContentMD     string             `json:"content_md"`
	Summary       *string            `json:"summary"`
	Status        string             `json:"status"`
	IsStarred     bool               `json:"is_starred"`
	ViewCount     int                `json:"view_count"`
	Tags          []TagResponse      `json:"tags"`
	Solutions     []SolutionResponse `json:"solutions"`
	SolutionCount int                `json:"solution_count"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

func NewNoteListItem(n *model.Note) NoteListItem {
	return NoteListItem{
		ID:            n.ID,
		ProblemID:     n.ProblemID,
		Problem:       NewProblemBrief(n.Problem),
		Title:         n.Title,
		Summary:       n.Summary,
		Status:        n.Status,
		IsStarred:     n.IsStarred,
		Tags:          NewTagResponses(n.Tags),
		SolutionCount: n.SolutionCount,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}

func NewNoteListItems(notes []*model.Note) []NoteListItem {
	out := make([]NoteListItem, 0, len(notes))
	for _, n := range notes {
		out = append(out, NewNoteListItem(n))
	}
	return out
}

func NewNoteResponse(n *model.Note) NoteResponse {
	count := n.SolutionCount
	if len(n.Solutions) > 0 {
		count = len(n.Solutions)
	}

	return NoteResponse{
		ID:            n.ID,
		ProblemID:     n.ProblemID,
		Problem:       NewProblemBrief(n.Problem),
		Title:         n.Title,
		ContentMD:     n.ContentMD,
		Summary:       n.Summary,
		Status:        n.Status,
		IsStarred:     n.IsStarred,
		ViewCount:     n.ViewCount,
		Tags:          NewTagResponses(n.Tags),
		Solutions:     NewSolutionResponses(n.Solutions),
		SolutionCount: count,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}
