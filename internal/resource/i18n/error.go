package i18n

import (
	"encoding/json"
	"strings"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-core/errorsx"
)

// LocalizeError 本地化结构化错误，同时保留错误码、原因和原始错误。
func LocalizeError(catalog *I18n, localeValue string, fallbackLocale string, err error) error {
	if err == nil {
		return nil
	}
	structured := normalizeStructuredError(kratosErrors.FromError(err))
	messageKey := structured.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY]
	if messageKey == "" {
		messageKey = defaultMessageKey(structured.Reason)
	} else if strings.HasPrefix(messageKey, "legacy.error.") && !catalog.HasMessage(messageKey) {
		if sourceKey, ok := catalog.KeyForSource(structured.Message); ok {
			messageKey = sourceKey
		} else {
			messageKey = defaultMessageKey(structured.Reason)
		}
	}
	messageArgs := decodeMessageArgs(structured.Metadata[errorsx.METADATA_KEY_MESSAGE_ARGS])
	localized := catalog.Localize(localeValue, fallbackLocale, messageKey, messageArgs, structured.Message)
	result := kratosErrors.Clone(structured)
	result.Message = localized
	metadata := make(map[string]string, len(result.Metadata)+2)
	for key, value := range result.Metadata {
		metadata[key] = value
	}
	metadata[errorsx.METADATA_KEY_MESSAGE_KEY] = messageKey
	if metadata[errorsx.METADATA_KEY_MESSAGE_ARGS] == "" {
		metadata[errorsx.METADATA_KEY_MESSAGE_ARGS] = "{}"
	}
	return result.WithMetadata(metadata).WithCause(err)
}

func normalizeStructuredError(err *kratosErrors.Error) *kratosErrors.Error {
	result := kratosErrors.Clone(err)
	switch result.Reason {
	case errorsx.ReasonInvalidArgument, errorsx.ReasonUnauthenticated, errorsx.ReasonPermissionDenied,
		errorsx.ReasonResourceNotFound, errorsx.ReasonConflict, errorsx.ReasonInternalError:
		return result
	}
	switch result.Code {
	case 400:
		result.Reason = errorsx.ReasonInvalidArgument
	case 401:
		result.Reason = errorsx.ReasonUnauthenticated
	case 403:
		result.Reason = errorsx.ReasonPermissionDenied
	case 404:
		result.Reason = errorsx.ReasonResourceNotFound
	case 409:
		result.Reason = errorsx.ReasonConflict
	default:
		result.Code = 500
		result.Reason = errorsx.ReasonInternalError
	}
	return result
}

func defaultMessageKey(reason string) string {
	switch reason {
	case errorsx.ReasonInvalidArgument:
		return "common.error.invalid_argument"
	case errorsx.ReasonUnauthenticated:
		return "common.error.unauthenticated"
	case errorsx.ReasonPermissionDenied:
		return "common.error.permission_denied"
	case errorsx.ReasonResourceNotFound:
		return "common.error.resource_not_found"
	case errorsx.ReasonConflict:
		return "common.error.conflict"
	default:
		return "common.error.internal"
	}
}

func decodeMessageArgs(value string) map[string]any {
	if value == "" {
		return nil
	}
	args := make(map[string]any)
	err := json.Unmarshal([]byte(value), &args)
	if err != nil {
		return nil
	}
	return args
}
