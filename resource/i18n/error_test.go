package i18n

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLocalizeErrorFallback(t *testing.T) {
	catalog, err := NewI18n("core", Assets())
	if err != nil {
		t.Fatal(err)
	}

	business := errors.FromError(LocalizeError(catalog, "zh-CN", "zh-CN", errorsx.InvalidArgument("未配置业务错误词条")))
	if business.Message != "未配置业务错误词条" {
		t.Fatalf("business error message = %q", business.Message)
	}
	if _, ok := business.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY]; ok {
		t.Fatal("business error should not expose an unstable message key")
	}

	internal := errors.FromError(LocalizeError(catalog, "zh-CN", "zh-CN", errorsx.Internal("内部实现细节")))
	if internal.Message != "系统内部错误" {
		t.Fatalf("internal error message = %q", internal.Message)
	}
	if internal.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != "common.error.internal" {
		t.Fatalf("internal error message key = %q", internal.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY])
	}
}

// TestLocalizeRateLimitedError 验证限流错误保持 429 并按请求语言显示明确提示。
func TestLocalizeRateLimitedError(t *testing.T) {
	catalog, err := NewI18n("core", Assets())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		locale string
		want   string
	}{
		{locale: "zh-CN", want: "请求过于频繁，请稍后重试"},
		{locale: "en-US", want: "Too many requests. Please try again later."},
		{locale: "ja-JP", want: "リクエストが多すぎます。しばらくしてからもう一度お試しください。"},
		{locale: "zh-TW", want: "請求過於頻繁，請稍後再試。"},
	}
	for _, test := range tests {
		localized := errors.FromError(LocalizeError(catalog, test.locale, "zh-CN", status.Error(codes.ResourceExhausted, "rate limit exceeded")))
		if localized.Code != 429 || localized.Reason != errorsx.ReasonRateLimited {
			t.Fatalf("localized error = (%d, %q), want (429, %q)", localized.Code, localized.Reason, errorsx.ReasonRateLimited)
		}
		if localized.Message != test.want {
			t.Fatalf("localized message for %s = %q, want %q", test.locale, localized.Message, test.want)
		}
		if localized.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY] != "common.error.rate_limited" {
			t.Fatalf("message key = %q, want common.error.rate_limited", localized.Metadata[errorsx.METADATA_KEY_MESSAGE_KEY])
		}
	}

	localized := errors.FromError(LocalizeError(catalog, "zh-CN", "zh-CN", status.Error(codes.Unavailable, "rate limit store unavailable")))
	if localized.Code != 500 || localized.Reason != errorsx.ReasonInternalError || localized.Message != "系统内部错误" {
		t.Fatalf("dependency error = (%d, %q, %q), want generic internal error", localized.Code, localized.Reason, localized.Message)
	}

	limited := errors.FromError(LocalizeError(catalog, "zh-CN", "zh-CN", errorsx.RateLimited("请求过于频繁，请稍后重试")))
	if limited.Code != 429 || limited.Reason != errorsx.ReasonRateLimited || limited.Message != "请求过于频繁，请稍后重试" {
		t.Fatalf("structured rate limit error = (%d, %q, %q)", limited.Code, limited.Reason, limited.Message)
	}
}
