package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

// paramInt64 解析路径参数里的正整数 ID。
//
// 统一在这里校验，避免每个 handler 各写一遍（也避免漏掉负数/非法值的判断）。
func paramInt64(c *gin.Context, name string) (int64, bool) {
	raw := c.Param(name)

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Fail(c, errs.ErrBadRequest.
			WithMessage("路径参数 "+name+" 必须是正整数"))
		return 0, false
	}
	return id, true
}
