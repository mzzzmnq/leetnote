package model

import "time"

// 题目难度。用常量而不是裸字符串，避免各处拼写不一致。
const (
	DifficultyEasy   = "Easy"
	DifficultyMedium = "Medium"
	DifficultyHard   = "Hard"
)

// DifficultyValues 用于校验（对应数据库上的 CHECK 约束）。
var DifficultyValues = []string{DifficultyEasy, DifficultyMedium, DifficultyHard}

// Problem 对应 problems 表 —— 全局共享的题目元数据。
//
// 注意它是【全局】的，不属于任何用户：题目本身与谁在刷题无关。
// 用户维度的数据在 notes 表里。
type Problem struct {
	ID         int64
	LeetCodeID *int // LeetCode 题号，手动录入的题可能没有
	Title      string
	TitleSlug  string // 如 two-sum，唯一，用于幂等导入
	Difficulty string
	URL        *string
	CreatedAt  time.Time

	// Tags 是题目所属的专题/知识点（按需填充，列表与详情都会带）
	Tags []*Tag
}
