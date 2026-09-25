package dto

import (
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/model"
)

type CreateTagInput struct {
	Name string `json:"name" binding:"required,max=50"`
	Slug string `json:"slug" binding:"required,max=50"`
	Kind string `json:"kind" binding:"required,oneof=algorithm data_structure topic"`
}

type TagResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

func NewTagResponse(t *model.Tag) TagResponse {
	if t == nil {
		return TagResponse{}
	}
	return TagResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Kind:      t.Kind,
		CreatedAt: t.CreatedAt,
	}
}

func NewTagResponses(tags []*model.Tag) []TagResponse {
	out := make([]TagResponse, 0, len(tags))
	for _, t := range tags {
		out = append(out, NewTagResponse(t))
	}
	return out
}
