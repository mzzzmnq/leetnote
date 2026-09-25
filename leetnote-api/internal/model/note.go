package model

import "time"

// 笔记状态。
const (
	NoteStatusDraft     = "draft"
	NoteStatusPublished = "published"
)

var NoteStatusValues = []string{NoteStatusDraft, NoteStatusPublished}

// Solution 对应 solutions 表 —— 一篇笔记可以有多个解法。
//
// 独立成表而不是塞进 JSON 字段的原因：需要按语言、复杂度筛选排序，
// 独立成表才能建索引。
type Solution struct {
	ID       int64
	NoteID   int64
	Title    string // 如「哈希表 · 一次遍历」
	Language string // python / go / java ...
	Code     string

	TimeComplexity  *string // O(n)
	SpaceComplexity *string // O(1)

	SortOrder int // 展示顺序，由用户决定哪个是「最优解」
	CreatedAt time.Time
}

// Note 对应 notes 表 —— 用户的核心内容。
//
// Problem / Tags / Solutions 是【关联数据】，只在需要时填充
// （列表页不填，详情页才填），避免列表查询产生 N+1。
type Note struct {
	ID        int64
	UserID    int64
	ProblemID *int64 // 可为空：支持「算法专题总结」这类不绑定题目的笔记
	Title     string
	ContentMD string
	Summary   *string
	Status    string
	IsStarred bool
	ViewCount int
	CreatedAt time.Time
	UpdatedAt time.Time

	// ---- 关联数据（按需填充）----
	Problem   *Problem
	Tags      []*Tag
	Solutions []*Solution

	// SolutionCount 用于列表页展示「N 个解法」，避免为计数去查 solutions 表
	SolutionCount int
}
