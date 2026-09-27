package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// dueQuery 是待复习列表的查询参数。
type dueQuery struct {
	Limit *int `form:"limit" binding:"omitempty,min=1,max=100"`
}

// Due GET /api/v1/reviews/due?limit=20
func (h *ReviewHandler) Due(c *gin.Context) {
	var q dueQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	limit := 20
	if q.Limit != nil {
		limit = *q.Limit
	}

	cards, err := h.svc.Due(c.Request.Context(), middleware.MustUserID(c), limit)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.NewReviewCardResponses(cards))
}

// Submit POST /api/v1/reviews/:id/submit
//
// 提交一次自评（0-5），服务端按 SM-2 算出新的间隔并落库。
func (h *ReviewHandler) Submit(c *gin.Context) {
	cardID, ok := paramInt64(c, "id")
	if !ok {
		return
	}

	var in dto.SubmitReviewInput
	if !validator.BindJSON(c, &in) {
		return
	}

	// 校验已在 binding 里做过（required 保证非 nil），这里可以安全解引用
	card, err := h.svc.Submit(c.Request.Context(), middleware.MustUserID(c), cardID, in)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.OK(c, dto.NewReviewCardResponse(card))
}

// Stats GET /api/v1/reviews/stats
func (h *ReviewHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewReviewStatsResponse(stats))
}
