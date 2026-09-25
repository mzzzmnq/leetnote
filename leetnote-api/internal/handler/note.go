package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type NoteHandler struct {
	svc *service.NoteService
}

func NewNoteHandler(svc *service.NoteService) *NoteHandler {
	return &NoteHandler{svc: svc}
}

// Create POST /api/v1/notes
func (h *NoteHandler) Create(c *gin.Context) {
	var in dto.NoteInput
	if !validator.BindJSON(c, &in) {
		return
	}

	note, err := h.svc.Create(c.Request.Context(), middleware.MustUserID(c), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, dto.NewNoteResponse(note))
}

// List GET /api/v1/notes
func (h *NoteHandler) List(c *gin.Context) {
	var q dto.NoteQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	notes, total, err := h.svc.List(c.Request.Context(), middleware.MustUserID(c), q)
	if err != nil {
		response.Fail(c, err)
		return
	}

	page, size := q.PageQuery.Normalize()
	response.Page(c, dto.NewNoteListItems(notes), total, page, size)
}

// Get GET /api/v1/notes/:id
//
// 越权访问（别人的笔记）会得到 404 而不是 403 —— 这样不会泄漏
// 「这个 ID 确实存在」。仓储层的 WHERE 已强制带上 user_id。
func (h *NoteHandler) Get(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	note, err := h.svc.GetByID(c.Request.Context(), middleware.MustUserID(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewNoteResponse(note))
}

// Update PUT /api/v1/notes/:id
func (h *NoteHandler) Update(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	var in dto.NoteInput
	if !validator.BindJSON(c, &in) {
		return
	}

	note, err := h.svc.Update(c.Request.Context(), middleware.MustUserID(c), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewNoteResponse(note))
}

// Delete DELETE /api/v1/notes/:id
func (h *NoteHandler) Delete(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), middleware.MustUserID(c), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// ToggleStar POST /api/v1/notes/:id/star
func (h *NoteHandler) ToggleStar(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	note, err := h.svc.ToggleStar(c.Request.Context(), middleware.MustUserID(c), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewNoteListItem(note))
}

// ListSolutions GET /api/v1/notes/:id/solutions
func (h *NoteHandler) ListSolutions(c *gin.Context) {
	noteID, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	solutions, err := h.svc.ListSolutions(c.Request.Context(), middleware.MustUserID(c), noteID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewSolutionResponses(solutions))
}

// CreateSolution POST /api/v1/notes/:id/solutions
func (h *NoteHandler) CreateSolution(c *gin.Context) {
	noteID, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	var in dto.SolutionInput
	if !validator.BindJSON(c, &in) {
		return
	}

	sol, err := h.svc.CreateSolution(c.Request.Context(), middleware.MustUserID(c), noteID, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, dto.NewSolutionResponse(sol))
}

// UpdateSolution PUT /api/v1/solutions/:id
func (h *NoteHandler) UpdateSolution(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	var in dto.SolutionInput
	if !validator.BindJSON(c, &in) {
		return
	}

	sol, err := h.svc.UpdateSolution(c.Request.Context(), middleware.MustUserID(c), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewSolutionResponse(sol))
}

// DeleteSolution DELETE /api/v1/solutions/:id
func (h *NoteHandler) DeleteSolution(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteSolution(c.Request.Context(), middleware.MustUserID(c), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
