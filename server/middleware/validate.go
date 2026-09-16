package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/liujitcn/kratos-core/errorsx"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"github.com/go-kratos/kratos/v3/middleware"
	"google.golang.org/protobuf/proto"
)

// NewValidateMiddleware 创建基于 Proto 声明的请求参数校验中间件。
func NewValidateMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if message, ok := req.(proto.Message); ok {
				if err := protovalidate.Validate(message); err != nil {
					return nil, validationError(err)
				}
			}
			return handler(ctx, req)
		}
	}
}

// validationError 将 protovalidate 的首条违规转换为统一业务错误。
func validationError(err error) error {
	if validationErr, ok := errors.AsType[*protovalidate.ValidationError](err); ok && len(validationErr.Violations) > 0 {
		violation := validationErr.Violations[0].Proto
		message := violation.GetMessage()
		if message == "" {
			message = "请求参数错误"
		}
		messageKey := standardValidationMessageKey(violation.GetRuleId(), violation.GetRule().GetElements())
		messageArgs := map[string]string{"Field": validationFieldPath(violation.GetField().GetElements())}
		return errorsx.WithMessageKey(errorsx.InvalidArgument(message), messageKey, messageArgs).WithCause(err)
	}
	return errorsx.WithMessageKey(errorsx.InvalidArgument("请求参数错误"), "common.error.invalid_argument", nil).WithCause(err)
}

// standardValidationMessageKey 将标准 Proto 校验规则映射到公共消息键。
func standardValidationMessageKey(ruleID string, elements []*validate.FieldPathElement) string {
	if len(elements) > 0 && elements[len(elements)-1].GetFieldName() == "required" {
		return "common.validation.required"
	}
	if isStandardValidationRule(ruleID, elements) {
		return "common.validation.invalid"
	}
	if ruleID != "" {
		return ruleID
	}
	return "common.validation.invalid"
}

// isStandardValidationRule 判断规则是否由 Protovalidate 内置规则生成。
func isStandardValidationRule(ruleID string, elements []*validate.FieldPathElement) bool {
	if len(elements) > 0 {
		switch elements[0].GetFieldName() {
		case "bytes", "double", "duration", "enum", "fixed32", "fixed64", "float", "int32", "int64", "map", "message", "repeated", "required", "sfixed32", "sfixed64", "sint32", "sint64", "string", "timestamp", "uint32", "uint64":
			return true
		}
	}
	for _, prefix := range []string{
		"bytes.", "double.", "duration.", "enum.", "fixed32.", "fixed64.", "float.", "int32.", "int64.", "map.", "message.", "repeated.", "required.", "sfixed32.", "sfixed64.", "sint32.", "sint64.", "string.", "timestamp.", "uint32.", "uint64.",
	} {
		if strings.HasPrefix(ruleID, prefix) {
			return true
		}
	}
	return ruleID == "required"
}

// validationFieldPath 将 Proto 字段路径转换为可展示的稳定路径。
func validationFieldPath(elements []*validate.FieldPathElement) string {
	fields := make([]string, 0, len(elements))
	for _, element := range elements {
		if element.GetFieldName() != "" {
			fields = append(fields, element.GetFieldName())
		}
	}
	if len(fields) == 0 {
		return "request"
	}
	return strings.Join(fields, ".")
}
