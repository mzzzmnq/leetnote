package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

type SolutionInput struct {
	Title           string  `json:"title" binding:"required,max=100"`
	Language        string  `json:"language" binding:"required,max=20"`
	Code            string  `json:"code" binding:"required,max=50000"`
	TimeComplexity  *string `json:"time_complexity" binding:"omitempty,max=50"`
	SpaceComplexity *string `json:"space_complexity" binding:"omitempty,max=50"`
}

type SolutionResponse struct {
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Language        string    `json:"language"`
	Code            string    `json:"code"`
	TimeComplexity  *string   `json:"time_complexity"`
	SpaceComplexity *string   `json:"space_complexity"`
	SortOrder       int       `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
}

func NewSolutionResponse(s *model.Solution) SolutionResponse {
	if s == nil {
		return SolutionResponse{}
	}
	return SolutionResponse{
		ID:              s.ID,
		Title:           s.Title,
		Language:        s.Language,
		Code:            s.Code,
		TimeComplexity:  s.TimeComplexity,
		SpaceComplexity: s.SpaceComplexity,
		SortOrder:       s.SortOrder,
		CreatedAt:       s.CreatedAt,
	}
}

func NewSolutionResponses(list []*model.Solution) []SolutionResponse {
	out := make([]SolutionResponse, 0, len(list))
	for _, s := range list {
		out = append(out, NewSolutionResponse(s))
	}
	return out
}
