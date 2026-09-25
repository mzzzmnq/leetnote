package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type TagHandler struct {
	svc *service.TagService
}

func NewTagHandler(svc *service.TagService) *TagHandler {
	return &TagHandler{svc: svc}
}

// tagListQuery 是标签列表的筛选参数。
type tagListQuery struct {
	Kind string `form:"kind" binding:"omitempty,oneof=algorithm data_structure topic"`
}

// Create POST /api/v1/tags
func (h *TagHandler) Create(c *gin.Context) {
	var in dto.CreateTagInput
	if !validator.BindJSON(c, &in) {
		return
	}

	t, err := h.svc.Create(c.Request.Context(), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, dto.NewTagResponse(t))
}

// List GET /api/v1/tags
//
// 标签总量很小（几十条），一次性返回即可，不做分页。
func (h *TagHandler) List(c *gin.Context) {
	var q tagListQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	tags, err := h.svc.List(c.Request.Context(), q.Kind)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewTagResponses(tags))
}

// Delete DELETE /api/v1/tags/:id
func (h *TagHandler) Delete(c *gin.Context) {
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
