package errorsx

import "github.com/go-kratos/kratos/v3/errors"

const (
	// ReasonInvalidArgument 表示请求参数错误。
	ReasonInvalidArgument = "INVALID_ARGUMENT"
	// ReasonUnauthenticated 表示认证失败。
	ReasonUnauthenticated = "UNAUTHENTICATED"
	// ReasonPermissionDenied 表示权限不足。
	ReasonPermissionDenied = "PERMISSION_DENIED"
	// ReasonResourceNotFound 表示资源不存在。
	ReasonResourceNotFound = "RESOURCE_NOT_FOUND"
	// ReasonConflict 表示资源状态冲突。
	ReasonConflict = "CONFLICT"
	// ReasonInternalError 表示服务内部错误。
	ReasonInternalError = "INTERNAL_ERROR"
)

// InvalidArgument 构造请求参数错误。
func InvalidArgument(message string) *errors.Error {
	return newStructuredError(400, ReasonInvalidArgument, message)
}

// Unauthenticated 构造认证失败错误。
func Unauthenticated(message string) *errors.Error {
	return newStructuredError(401, ReasonUnauthenticated, message)
}

// PermissionDenied 构造权限不足错误。
func PermissionDenied(message string) *errors.Error {
	return newStructuredError(403, ReasonPermissionDenied, message)
}

// ResourceNotFound 构造资源不存在错误。
func ResourceNotFound(message string) *errors.Error {
	return newStructuredError(404, ReasonResourceNotFound, message)
}

// Conflict 构造状态冲突错误。
func Conflict(message string) *errors.Error {
	return newStructuredError(409, ReasonConflict, message)
}

// Internal 构造内部错误。
func Internal(message string) *errors.Error {
	return newStructuredError(500, ReasonInternalError, message)
}

// newStructuredError 构造携带兼容消息键的结构化错误。
func newStructuredError(code int, reason, message string) *errors.Error {
	structuredErr := errors.New(code, reason, message)
	if message == "" {
		return structuredErr
	}
	return WithMessageKey(structuredErr, MessageKey(message), nil)
}
