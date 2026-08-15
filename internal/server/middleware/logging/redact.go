package logging

import (
	"encoding/json"
	"strings"
)

const logRedactedValue = "***REDACTED***"

var sensitiveLogKeyExactSet = map[string]struct{}{
	"authorization":      {},
	"proxyauthorization": {},
	"cookie":             {},
	"setcookie":          {},
	"password":           {},
	"passwd":             {},
	"pwd":                {},
	"secret":             {},
	"apikey":             {},
	"privatekey":         {},
	"accesskey":          {},
	"datastore":          {},
	"cachestore":         {},
	"dsn":                {},
	"connectionstring":   {},
}

// redactLogHeaderMap 对访问日志请求头执行脱敏处理。
func redactLogHeaderMap(headers map[string]string) map[string]string {
	for key, value := range headers {
		if isSensitiveLogKey(key) || isSensitiveLogString(value) {
			headers[key] = logRedactedValue
		}
	}
	return headers
}

// redactLogJSON 对日志 JSON 内容执行递归脱敏处理。
func redactLogJSON(body []byte) []byte {
	var payload interface{}
	err := json.Unmarshal(body, &payload)
	if err != nil {
		return body
	}

	payload = redactLogValue("", payload)
	var redactedBody []byte
	redactedBody, err = json.Marshal(payload)
	if err != nil {
		return body
	}
	return redactedBody
}

// redactLogValue 根据字段名和字段值递归脱敏敏感日志内容。
func redactLogValue(key string, value interface{}) interface{} {
	if isSensitiveLogKey(key) {
		return logRedactedValue
	}

	switch typed := value.(type) {
	case map[string]interface{}:
		for childKey, childValue := range typed {
			typed[childKey] = redactLogValue(childKey, childValue)
		}
		return typed
	case []interface{}:
		for index, item := range typed {
			typed[index] = redactLogValue("", item)
		}
		return typed
	case string:
		if isSensitiveLogString(typed) {
			return logRedactedValue
		}
		return typed
	default:
		return typed
	}
}

// isSensitiveLogKey 判断字段名是否属于日志脱敏范围。
func isSensitiveLogKey(key string) bool {
	normalizedKey := normalizeSensitiveLogKey(key)
	if normalizedKey == "" {
		return false
	}
	if _, ok := sensitiveLogKeyExactSet[normalizedKey]; ok {
		return true
	}
	if strings.HasSuffix(normalizedKey, "password") ||
		strings.HasSuffix(normalizedKey, "passwd") ||
		strings.HasSuffix(normalizedKey, "secret") ||
		strings.HasSuffix(normalizedKey, "apikey") ||
		strings.HasSuffix(normalizedKey, "privatekey") ||
		strings.HasSuffix(normalizedKey, "accesskey") ||
		strings.HasSuffix(normalizedKey, "token") {
		return true
	}
	return false
}

// normalizeSensitiveLogKey 归一化日志字段名便于识别敏感字段。
func normalizeSensitiveLogKey(key string) string {
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", " ", "")
	return strings.ToLower(replacer.Replace(key))
}

// isSensitiveLogString 判断字符串值是否疑似包含密钥、令牌或数据库连接信息。
func isSensitiveLogString(value string) bool {
	normalizedValue := strings.ToLower(value)
	if normalizedValue == "" {
		return false
	}
	if strings.HasPrefix(normalizedValue, "bearer ") ||
		strings.HasPrefix(normalizedValue, "basic ") ||
		strings.HasPrefix(normalizedValue, "sk-") ||
		strings.Contains(normalizedValue, "mysql://") ||
		strings.Contains(normalizedValue, "postgres://") ||
		strings.Contains(normalizedValue, "@tcp(") {
		return true
	}
	return false
}
