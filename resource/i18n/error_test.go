package i18n

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/kratos-core/errorsx"
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
