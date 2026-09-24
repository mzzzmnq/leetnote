package errs

import (
	"fmt"
	"net/http"
)

// AppError 是可携带「业务错误码 + HTTP 状态码」的错误类型。
//
// 设计要点（面试会问）：
//   - Code 是给前端/调用方做分支判断的稳定标识，不要用 Message 判断。
//   - Message 是给用户看的中文提示，可以改；Code 不能随便改。
//   - Err 保存底层原始错误（如 SQL 报错），只写日志，绝不返回给前端——
//     否则会泄漏表结构等内部信息。
type AppError struct {
	Code       string // 机器可读的错误码，如 NOTE_NOT_FOUND
	Message    string // 面向用户的中文提示
	HTTPStatus int    // HTTP 状态码
	Details    any    // 可选的结构化补充信息（如字段级校验错误）
	Err        error  // 底层原始错误，仅用于日志
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 让 errors.Is / errors.As 能穿透到底层错误。
func (e *AppError) Unwrap() error { return e.Err }

// Is 让 errors.Is 按【错误码】比较，而不是按指针。
//
// 为什么必须有：ErrNotFound 这类预定义变量是全局共享的，
// Wrap / WithMessage 会派生出新对象。若不实现 Is，
// errors.Is(derived, errs.ErrNotFound) 会因为指针不同而返回 false，
// 调用方就再也判断不出错误类型了。
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Wrap 返回一个附带了底层错误的副本（不修改原对象，避免全局变量被污染）。
func (e *AppError) Wrap(err error) *AppError {
	clone := *e
	clone.Err = err
	return &clone
}

// WithDetails 返回一个附带了结构化信息的副本。
func (e *AppError) WithDetails(d any) *AppError {
	clone := *e
	clone.Details = d
	return &clone
}

// WithMessage 返回一个替换了提示文案的副本。
func (e *AppError) WithMessage(msg string) *AppError {
	clone := *e
	clone.Message = msg
	return &clone
}

// New 构造一个自定义错误。
func New(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

// NotFound 构造资源级 404，如 NotFound("NOTE", "笔记") → code=NOTE_NOT_FOUND。
func NotFound(code, resource string) *AppError {
	return &AppError{
		Code:       code + "_NOT_FOUND",
		Message:    resource + "不存在",
		HTTPStatus: http.StatusNotFound,
	}
}

// 预定义的通用错误。它们是只读模板，使用时务必用 Wrap / WithMessage 派生副本。
var (
	ErrInternal     = &AppError{Code: "INTERNAL_ERROR", Message: "服务器内部错误", HTTPStatus: http.StatusInternalServerError}
	ErrBadRequest   = &AppError{Code: "BAD_REQUEST", Message: "请求参数有误", HTTPStatus: http.StatusBadRequest}
	ErrValidation   = &AppError{Code: "VALIDATION_FAILED", Message: "参数校验未通过", HTTPStatus: http.StatusUnprocessableEntity}
	ErrUnauthorized = &AppError{Code: "UNAUTHORIZED", Message: "未登录或登录状态已过期", HTTPStatus: http.StatusUnauthorized}
	ErrForbidden    = &AppError{Code: "FORBIDDEN", Message: "没有权限执行该操作", HTTPStatus: http.StatusForbidden}
	ErrNotFound     = &AppError{Code: "NOT_FOUND", Message: "资源不存在", HTTPStatus: http.StatusNotFound}
	ErrConflict     = &AppError{Code: "CONFLICT", Message: "资源已存在", HTTPStatus: http.StatusConflict}
	ErrTooManyReqs  = &AppError{Code: "TOO_MANY_REQUESTS", Message: "请求过于频繁，请稍后再试", HTTPStatus: http.StatusTooManyRequests}

	ErrRouteNotFound = &AppError{Code: "ROUTE_NOT_FOUND", Message: "接口不存在", HTTPStatus: http.StatusNotFound}
)
