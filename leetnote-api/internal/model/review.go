package model

import "time"

// ReviewCard 对应 review_cards 表 —— 一篇笔记对应一张复习卡。
//
// 字段含义（SM-2 算法的状态）：
//   - EaseFactor  : 难度系数，越大说明越容易，间隔增长越快。下限 1.3
//   - IntervalDays: 当前间隔天数
//   - Repetitions : 连续答对次数，答错归零
//   - DueAt       : 下次该复习的时间
//
// 注意：这张表【没有】CreatedAt / UpdatedAt（见迁移 000001）。
type ReviewCard struct {
	ID          int64
	NoteID      int64
	NoteTitle   string // 关联查出来的，便于直接展示
	NoteSummary *string

	EaseFactor     float64
	IntervalDays   int
	Repetitions    int
	DueAt          time.Time
	LastReviewedAt *time.Time
}

// ReviewLog 对应 review_logs 表 —— 每次复习的流水。
//
// 设计成「只追加不修改」的审计日志：可以回溯复习曲线，
// 也便于做统计图表，而且出问题时有据可查。
type ReviewLog struct {
	ID           int64
	CardID       int64
	Rating       int // 0-5
	PrevInterval int
	NextInterval int
	ReviewedAt   time.Time
}

// ReviewStats 是复习统计。
type ReviewStats struct {
	// 待复习（due_at <= now）
	DueCount int64
	// 卡片总数
	TotalCards int64
	// 已复习过至少一次的卡片数
	LearnedCards int64
	// 累计复习次数
	TotalReviews int64
	// 未来 7 天每天到期的数量（含今天），用于画「复习压力」柱状图
	Forecast []ForecastPoint
}

// ForecastPoint 是复习计划里的一个点。
type ForecastPoint struct {
	Date  time.Time
	Count int64
}
