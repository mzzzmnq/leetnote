package dto

import "github.com/mzzzmnq/leetnote-api/internal/ai"

// SimilarNoteResponse 是「相似题推荐」里的一条。
//
// 它直接来自 AI 服务，不是数据库实体，所以不复用 NoteListItem。
// 字段刻意保持精简：推荐位只需要能点进去的最少信息。
type SimilarNoteResponse struct {
	NoteID     int64   `json:"note_id"`
	Title      string  `json:"title"`
	Summary    *string `json:"summary"`
	IsStarred  bool    `json:"is_starred"`
	Similarity float64 `json:"similarity"`
}

// SimilarNotesResponse 是相似题推荐接口的响应。
type SimilarNotesResponse struct {
	NoteID int64                 `json:"note_id"`
	Model  string                `json:"model"`
	Items  []SimilarNoteResponse `json:"items"`
}

func NewSimilarNotesResponse(noteID int64, model string, items []ai.SimilarNote) SimilarNotesResponse {
	out := make([]SimilarNoteResponse, 0, len(items))
	for _, item := range items {
		out = append(out, SimilarNoteResponse{
			NoteID:     item.NoteID,
			Title:      item.Title,
			Summary:    item.Summary,
			IsStarred:  item.IsStarred,
			Similarity: item.Similarity,
		})
	}

	return SimilarNotesResponse{NoteID: noteID, Model: model, Items: out}
}
