package dto

import "github.com/mzzzmnq/leetnote-api/internal/model"

// DifficultyBreakdown 是按难度的分布。
//
// 单独嵌套一层而不是拍平，是为了让前端直接把它喂给 ECharts 的饼图，
// 不用再手动拼数据。
type DifficultyBreakdown struct {
	Easy   int64 `json:"easy"`
	Medium int64 `json:"medium"`
	Hard   int64 `json:"hard"`
}

type StatsOverviewResponse struct {
	TotalNotes     int64               `json:"total_notes"`
	TotalProblems  int64               `json:"total_problems"`
	TotalSolutions int64               `json:"total_solutions"`
	TotalTags      int64               `json:"total_tags"`
	Stars          int64               `json:"starred"`
	Drafts         int64               `json:"drafts"`
	ActiveDays     int64               `json:"active_days"`
	CurrentStreak  int                 `json:"current_streak"`
	LongestStreak  int                 `json:"longest_streak"`
	Difficulty     DifficultyBreakdown `json:"difficulty"`
}

func NewStatsOverviewResponse(s *model.StatsOverview) StatsOverviewResponse {
	return StatsOverviewResponse{
		TotalNotes:     s.TotalNotes,
		TotalProblems:  s.TotalProblems,
		TotalSolutions: s.TotalSolutions,
		TotalTags:      s.TotalTags,
		Stars:          s.Starred,
		Drafts:         s.Drafts,
		ActiveDays:     s.ActiveDays,
		CurrentStreak:  s.CurrentStreak,
		LongestStreak:  s.LongestStreak,
		Difficulty: DifficultyBreakdown{
			Easy:   s.Easy,
			Medium: s.Medium,
			Hard:   s.Hard,
		},
	}
}

// TrendPointResponse 是趋势图上的一个点。
type TrendPointResponse struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int64  `json:"count"`
}

type TrendResponse struct {
	Days   int                  `json:"days"`
	Points []TrendPointResponse `json:"points"`
}

func NewTrendResponse(days int, points []*model.TrendPoint) TrendResponse {
	out := make([]TrendPointResponse, 0, len(points))
	for _, p := range points {
		out = append(out, TrendPointResponse{
			Date:  p.Day.Format("2006-01-02"),
			Count: p.Count,
		})
	}
	return TrendResponse{Days: days, Points: out}
}
