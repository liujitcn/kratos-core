package errorsx

import (
	"errors"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"gorm.io/gorm"
)

// IsStructuredError 判断错误是否已经携带稳定 reason。
func IsStructuredError(err error) bool {
	var kratosErr *kratosErrors.Error
	// 已经是 Kratos 错误且 reason 非空时，视为已完成分类。
	return errors.As(err, &kratosErr) && kratosErr.Reason != ""
}

// WrapIfNeeded 在错误尚未完成分类时，使用兜底错误包装。
func WrapIfNeeded(err error, fallback *kratosErrors.Error) error {
	if err == nil {
		return nil
	}
	// 已经完成分类的错误直接透传，避免覆盖原始语义。
	if IsStructuredError(err) {
		return err
	}
	if fallback == nil {
		return err
	}
	return fallback.WithCause(err)
}

// WrapInternal 在错误尚未完成分类时，包装成内部错误。
func WrapInternal(err error, message string) error {
	// 仓储层统一返回 gorm.ErrRecordNotFound，这类可预期查询空结果应对外表现为资源不存在。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ResourceNotFound(message).WithCause(err)
	}
	return WrapIfNeeded(err, Internal(message))
}
