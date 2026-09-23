package errs_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
)

// 这条测试防的是一个很容易踩的坑：AppError 的预定义变量是全局共享的，
// 如果 Wrap/WithMessage 直接改原对象，就会污染所有后续请求。
func TestWrapDoesNotMutateOriginal(t *testing.T) {
	underlying := errors.New("sql: no rows in result set")

	wrapped := errs.ErrNotFound.Wrap(underlying)

	if errs.ErrNotFound.Err != nil {
		t.Fatal("Wrap 修改了全局错误对象，会造成跨请求污染")
	}
	if !errors.Is(wrapped, underlying) {
		t.Fatal("Unwrap 链路断裂，errors.Is 无法穿透到底层错误")
	}
	if wrapped.Code != errs.ErrNotFound.Code {
		t.Errorf("Wrap 后错误码变了: %s", wrapped.Code)
	}
	if wrapped.HTTPStatus != errs.ErrNotFound.HTTPStatus {
		t.Errorf("Wrap 后状态码变了: %d", wrapped.HTTPStatus)
	}
}

func TestWithMessageDoesNotMutateOriginal(t *testing.T) {
	got := errs.ErrConflict.WithMessage("用户名已被占用")

	if got.Message != "用户名已被占用" {
		t.Errorf("提示文案未替换: %s", got.Message)
	}
	if errs.ErrConflict.Message == "用户名已被占用" {
		t.Fatal("WithMessage 污染了全局错误对象")
	}
}

func TestWithDetails(t *testing.T) {
	got := errs.ErrValidation.WithDetails(map[string]string{"email": "格式不正确"})

	if got.Details == nil {
		t.Fatal("Details 未设置")
	}
	if errs.ErrValidation.Details != nil {
		t.Fatal("WithDetails 污染了全局错误对象")
	}
}

func TestNotFound(t *testing.T) {
	e := errs.NotFound("NOTE", "笔记")

	if e.Code != "NOTE_NOT_FOUND" {
		t.Errorf("错误码错误: 期望 NOTE_NOT_FOUND, 实际 %s", e.Code)
	}
	if e.Message != "笔记不存在" {
		t.Errorf("提示文案错误: %s", e.Message)
	}
	if e.HTTPStatus != http.StatusNotFound {
		t.Errorf("状态码错误: %d", e.HTTPStatus)
	}
}

func TestErrorString(t *testing.T) {
	if got := errs.ErrNotFound.Error(); got != "NOT_FOUND: 资源不存在" {
		t.Errorf("Error() 输出不符: %s", got)
	}

	wrapped := errs.ErrInternal.Wrap(errors.New("boom"))
	if got := wrapped.Error(); got != "INTERNAL_ERROR: 服务器内部错误: boom" {
		t.Errorf("带底层错误的 Error() 输出不符: %s", got)
	}
}
