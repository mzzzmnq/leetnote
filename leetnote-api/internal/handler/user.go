package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/dto"
	"github.com/mzzzmnq/leetnote-api/internal/middleware"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
	"github.com/mzzzmnq/leetnote-api/internal/service"
	"github.com/mzzzmnq/leetnote-api/internal/validator"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// Me 处理 GET /api/v1/users/me
//
// 注意：所有涉及「我的数据」的查询都必须带 user_id 条件，
// 绝不能只按资源 ID 查——那是典型的越权漏洞（IDOR）。
func (h *UserHandler) Me(c *gin.Context) {
	u, err := h.svc.GetByID(c.Request.Context(), middleware.MustUserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewUserResponse(u))
}

// UpdateMe 处理 PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var in dto.UpdateProfileInput
	if !validator.BindJSON(c, &in) {
		return
	}

	u, err := h.svc.UpdateProfile(c.Request.Context(), middleware.MustUserID(c), in)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, dto.NewUserResponse(u))
}

// ChangePassword 处理 POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var in dto.ChangePasswordInput
	if !validator.BindJSON(c, &in) {
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), middleware.MustUserID(c), in); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
