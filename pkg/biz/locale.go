package biz

import (
	"context"
	"strings"
)

type localeContextKey struct{}

// WithLocale 将当前请求语言写入上下文。
func WithLocale(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, localeContextKey{}, value)
}

// LocaleFromContext 从上下文读取当前请求语言。
func LocaleFromContext(ctx context.Context) string {
	value, ok := ctx.Value(localeContextKey{}).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
