package errorsx

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
)

const (
	// METADATA_KEY_MESSAGE_KEY 标识稳定的国际化消息键。
	METADATA_KEY_MESSAGE_KEY = "message_key"
	// METADATA_KEY_MESSAGE_ARGS 标识国际化消息命名参数。
	METADATA_KEY_MESSAGE_ARGS = "message_args"
)

// MessageKey 返回由固定源文生成的兼容消息键。
func MessageKey(message string) string {
	digest := sha256.Sum256([]byte(message))
	return "legacy.error." + hex.EncodeToString(digest[:8])
}

// WithMessageKey 为结构化错误补充稳定消息键和命名参数。
func WithMessageKey(structuredErr *kratosErrors.Error, messageKey string, messageArgs map[string]string) *kratosErrors.Error {
	if structuredErr == nil {
		return nil
	}
	metadata := make(map[string]string, len(structuredErr.Metadata)+2)
	for key, value := range structuredErr.Metadata {
		metadata[key] = value
	}
	metadata[METADATA_KEY_MESSAGE_KEY] = messageKey
	metadata[METADATA_KEY_MESSAGE_ARGS] = "{}"
	if len(messageArgs) > 0 {
		var data []byte
		var err error
		data, err = json.Marshal(messageArgs)
		if err == nil {
			metadata[METADATA_KEY_MESSAGE_ARGS] = string(data)
		}
	}
	return structuredErr.WithMetadata(metadata)
}
