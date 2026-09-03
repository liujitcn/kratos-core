package client

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/sdk"
)

type testKey struct {
	purposes []string
}

// Derive 返回测试用运行时密钥并记录用途。
func (k *testKey) Derive(_ context.Context, purpose string) ([]byte, error) {
	k.purposes = append(k.purposes, purpose)
	return []byte("runtime-jwt-key"), nil
}

func TestAppendClientMiddlewaresUsesSharedRuntimeJWTKey(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	runtimeKey := &testKey{}
	sdk.Runtime.SetKey(runtimeKey)
	t.Cleanup(func() {
		sdk.Runtime.SetKey(previousKey)
	})

	_, err := appendClientMiddlewaresWithMetadata(&configv1.Client_Middleware{Auth: &configv1.Client_Middleware_Auth{}}, nil, middleware.Middleware(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeKey.purposes) != 1 || runtimeKey.purposes[0] != "kratos-kit:authn/jwt" {
		t.Fatalf("unexpected runtime key purposes: %v", runtimeKey.purposes)
	}
}

func TestAppendClientMiddlewaresPrefersConfiguredJWTKey(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	runtimeKey := &testKey{}
	sdk.Runtime.SetKey(runtimeKey)
	t.Cleanup(func() {
		sdk.Runtime.SetKey(previousKey)
	})

	_, err := appendClientMiddlewaresWithMetadata(&configv1.Client_Middleware{Auth: &configv1.Client_Middleware_Auth{Secret: "configured-secret"}}, nil, middleware.Middleware(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeKey.purposes) != 0 {
		t.Fatalf("configured secret unexpectedly used runtime key: %v", runtimeKey.purposes)
	}
}
