package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

// SubmitReviewInput 是提交复习评分的请求体。
//
// Rating 用【指针】：0 是合法评分（完全没想起来），
// 如果用 int + required，0 会被 required 判为「没传」而报错；
// 如果去掉 required，漏传又会静默当成 0。指针才能区分这两者。
type SubmitReviewInput struct {
	Rating *int `json:"rating" binding:"required,min=0,max=5"`
}

type ReviewCardResponse struct {
	ID          int64   `json:"id"`
	NoteID      int64   `json:"note_id"`
	NoteTitle   string  `json:"note_title"`
	NoteSummary *string `json:"note_summary"`

	EaseFactor   float64 `json:"ease_factor"`
	IntervalDays int     `json:"interval_days"`
	Repetitions  int     `json:"repetitions"`

	DueAt          time.Time  `json:"due_at"`
	LastReviewedAt *time.Time `json:"last_reviewed_at"`
}

func NewReviewCardResponse(c *model.ReviewCard) ReviewCardResponse {
	return ReviewCardResponse{
		ID:             c.ID,
		NoteID:         c.NoteID,
		NoteTitle:      c.NoteTitle,
		NoteSummary:    c.NoteSummary,
		EaseFactor:     c.EaseFactor,
		IntervalDays:   c.IntervalDays,
		Repetitions:    c.Repetitions,
		DueAt:          c.DueAt,
		LastReviewedAt: c.LastReviewedAt,
	}
}

func NewReviewCardResponses(cards []*model.ReviewCard) []ReviewCardResponse {
	out := make([]ReviewCardResponse, 0, len(cards))
	for _, c := range cards {
		out = append(out, NewReviewCardResponse(c))
	}
	return out
}

// ForecastPointResponse 是复习计划里的一个点。
type ForecastPointResponse struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ReviewStatsResponse struct {
	DueCount     int64                   `json:"due_count"`
	TotalCards   int64                   `json:"total_cards"`
	LearnedCards int64                   `json:"learned_cards"`
	TotalReviews int64                   `json:"total_reviews"`
	Forecast     []ForecastPointResponse `json:"forecast"`
}

func NewReviewStatsResponse(s *model.ReviewStats) ReviewStatsResponse {
	forecast := make([]ForecastPointResponse, 0, len(s.Forecast))
	for _, p := range s.Forecast {
		forecast = append(forecast, ForecastPointResponse{
			Date:  p.Date.Format("2006-01-02"),
			Count: p.Count,
		})
	}

	return ReviewStatsResponse{
		DueCount:     s.DueCount,
		TotalCards:   s.TotalCards,
		LearnedCards: s.LearnedCards,
		TotalReviews: s.TotalReviews,
		Forecast:     forecast,
	}
}
