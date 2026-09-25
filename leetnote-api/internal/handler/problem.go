package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type ProblemHandler struct {
	svc *service.ProblemService
}

func NewProblemHandler(svc *service.ProblemService) *ProblemHandler {
	return &ProblemHandler{svc: svc}
}

// Create POST /api/v1/problems
func (h *ProblemHandler) Create(c *gin.Context) {
	var in dto.ProblemInput
	if !validator.BindJSON(c, &in) {
		return
	}

	p, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, dto.NewProblemResponse(p))
}

// List GET /api/v1/problems
func (h *ProblemHandler) List(c *gin.Context) {
	var q dto.ProblemQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	problems, total, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.Fail(c, err)
		return
	}

	page, size := q.PageQuery.Normalize()
	response.Page(c, dto.NewProblemResponses(problems), total, page, size)
}

// Get GET /api/v1/problems/:id
func (h *ProblemHandler) Get(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	p, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewProblemResponse(p))
}

// Update PUT /api/v1/problems/:id
func (h *ProblemHandler) Update(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	var in dto.ProblemInput
	if !validator.BindJSON(c, &in) {
		return
	}

	p, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewProblemResponse(p))
}

// Delete DELETE /api/v1/problems/:id
//
// 注意：删除题目不会删掉关联的笔记（外键是 ON DELETE SET NULL），
// 用户的笔记仍然保留，只是不再关联题目。
func (h *ProblemHandler) Delete(c *gin.Context) {
	id, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
