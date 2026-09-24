package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/mzzzmnq/leetnote-api/internal/pkg/errs"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/response"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,50}$`)

// Register 注册自定义校验规则与字段名解析。
//
// 必须在路由启动前调用一次。
func Register() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	// 让校验错误里用的是 JSON 字段名（username）而不是结构体字段名（Username），
	// 这样前端能直接把错误映射到表单控件上。
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	_ = v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return usernameRe.MatchString(fl.Field().String())
	})
}

// BindJSON 解析并校验请求体。
//
// 校验失败时自动写出 422 响应并返回 false，handler 直接 return 即可：
//
//	if !validator.BindJSON(c, &in) { return }
//
// 返回的错误形如：
//
//	{"error":{"code":"VALIDATION_FAILED","message":"参数校验未通过",
//	          "details":{"username":"用户名只能包含字母、数字、下划线，长度 3-50"}}}
func BindJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			details := make(map[string]string, len(ve))
			for _, fe := range ve {
				details[fe.Field()] = messageFor(fe)
			}
			response.Fail(c, errs.ErrValidation.WithDetails(details))
			return false
		}

		response.Fail(c, errs.ErrBadRequest.
			WithMessage("请求体不是合法的 JSON").
			Wrap(err))
		return false
	}
	return true
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "该字段必填"
	case "email":
		return "邮箱格式不正确"
	case "url":
		return "必须是合法的 URL"
	case "username":
		return "用户名只能包含字母、数字、下划线，长度 3-50"
	case "min":
		return fmt.Sprintf("长度不能少于 %s", fe.Param())
	case "max":
		return fmt.Sprintf("长度不能超过 %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("只能是以下值之一：%s", fe.Param())
	default:
		return "格式不正确"
	}
}
