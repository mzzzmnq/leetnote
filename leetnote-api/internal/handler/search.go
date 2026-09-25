package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type SearchHandler struct {
	svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// Search GET /api/v1/search?q=&type=&limit=
//
// 一次返回笔记与题目两类结果，前端分组展示。
func (h *SearchHandler) Search(c *gin.Context) {
	var q dto.SearchQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	result, err := h.svc.Search(c.Request.Context(), middleware.MustUserID(c), q)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.SearchResponse{
		Keyword:  q.Q,
		Notes:    dto.NewNoteListItems(result.Notes),
		Problems: dto.NewProblemResponses(result.Problems),
	})
}
