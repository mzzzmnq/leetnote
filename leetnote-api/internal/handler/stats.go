package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// trendQuery 是趋势接口的查询参数。
//
// Days 用【指针】而不是 int：gin 的 binding 里 `omitempty` 对数值类型
// 会跳过零值校验，导致 `?days=0` 被当成「没传」而绕过 min=1。
// 用 *int 就能区分「没传」（nil，用默认值）和「传了 0」（非法，返回 422）。
type trendQuery struct {
	Days *int `form:"days" binding:"omitempty,min=1,max=365"`
}

// Overview GET /api/v1/stats/overview
func (h *StatsHandler) Overview(c *gin.Context) {
	stats, err := h.svc.Overview(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewStatsOverviewResponse(stats))
}

// Trend GET /api/v1/stats/trend?days=30
func (h *StatsHandler) Trend(c *gin.Context) {
	var q trendQuery
	if !validator.BindQuery(c, &q) {
		return
	}

	days := 30
	if q.Days != nil {
		days = *q.Days
	}

	points, err := h.svc.Trend(c.Request.Context(), middleware.MustUserID(c), days)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewTrendResponse(days, points))
}
