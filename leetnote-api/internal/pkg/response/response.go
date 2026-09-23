package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// ---------- 响应结构 ----------

// ErrorDetail 是错误响应里的具体描述。
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ErrorBody 是统一的错误响应体：{ "error": { ... } }
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// PageData 是统一的分页响应体。
type PageData[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
	Pages int   `json:"pages"`
}

// ---------- 成功响应 ----------

// OK 返回 200 + 数据。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

// Created 返回 201 + 新资源。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

// NoContent 返回 204（如删除成功）。
//
// 注意：gin 的 c.Status() 只是记录状态码，真正写出需要 WriteHeaderNow()。
// 走完整路由时 engine 会自动刷写，但直接调用 handler 时不会，所以这里显式刷一次。
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// Page 返回统一的分页结构，自动计算总页数。
func Page[T any](c *gin.Context, items []T, total int64, page, size int) {
	pages := 0
	if size > 0 {
		pages = int((total + int64(size) - 1) / int64(size))
	}
	OK(c, PageData[T]{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
		Pages: pages,
	})
}

// ---------- 错误响应 ----------

// Fail 把任意 error 翻译成统一错误响应。
//
// 关键点：
//   - 用 errors.As 识别 *errs.AppError，拿到业务错误码与 HTTP 状态码
//   - 未识别的错误一律当 500 处理，且只返回笼统提示，避免泄漏内部细节
//   - 底层错误（appErr.Err）只写进日志
func Fail(c *gin.Context, err error) {
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		appErr = errs.ErrInternal.Wrap(err)
	}

	if appErr.Err != nil {
		slog.Error("request failed",
			"code", appErr.Code,
			"error", appErr.Err.Error(),
			"path", c.Request.URL.Path,
			"request_id", c.GetString("request_id"),
		)
	}

	c.AbortWithStatusJSON(appErr.HTTPStatus, ErrorBody{
		Error: ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}

// Abort 等同于 Fail，语义上更明确表示「中断后续 handler」。
func Abort(c *gin.Context, err error) {
	Fail(c, err)
}
