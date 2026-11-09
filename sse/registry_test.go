package sse

import (
	"context"
	"fmt"
	"testing"

	"github.com/liujitcn/kratos-kit/auth"
	"github.com/liujitcn/kratos-kit/auth/data"
)

// authorizedTestStream 用于验证自定义频道语义不会被核心层按字符串猜测。
type authorizedTestStream struct{}

// ID 返回测试流名称。
func (authorizedTestStream) ID() string { return "project" }

// Resolve 模拟旧解析路径，不能由 HTTP 认证订阅回落进入。
func (authorizedTestStream) Resolve(string, int64) (string, error) {
	return "unsafe-fallback", nil
}

// ResolveContext 验证可信身份传播以及合法跨租户授权。
func (authorizedTestStream) ResolveContext(ctx context.Context, channel string, identity *data.UserTokenPayload) (string, error) {
	current, err := auth.FromContext(ctx)
	if err != nil || current.UserId != identity.UserId {
		return "", fmt.Errorf("可信身份未传播")
	}
	if channel == "tenant-two:project-seven" && identity.RoleCode != "admin" {
		return "", fmt.Errorf("无权访问目标项目")
	}
	return "opaque-resource-key/" + channel, nil
}

// TestRegistryResourceAuthorization 验证跨租户管理保留，普通用户越权与匿名请求被拒绝。
func TestRegistryResourceAuthorization(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(authorizedTestStream{})
	if err != nil {
		t.Fatal(err)
	}
	identity := &data.UserTokenPayload{UserId: 1, TenantId: 1, RoleCode: "operator"}
	_, _, err = registry.ResolveContext(context.Background(), "project", "tenant-two:project-seven", identity)
	if err == nil {
		t.Fatal("普通用户跨租户订阅未被拒绝")
	}
	identity.RoleCode = "admin"
	var transportID string
	transportID, _, err = registry.ResolveContext(context.Background(), "project", "tenant-two:project-seven", identity)
	if err != nil || transportID != "opaque-resource-key/tenant-two:project-seven" {
		t.Fatalf("授权跨租户订阅失败: %s, %v", transportID, err)
	}
	_, _, err = registry.ResolveContext(context.Background(), "project", "any", nil)
	if err == nil {
		t.Fatal("匿名订阅未被拒绝")
	}
}

// deferredTestStream 模拟组合根尚未绑定业务客户端时的流定义。
type deferredTestStream struct{}

// ID 返回等待授权器绑定的流标识。
func (deferredTestStream) ID() string { return "deferred" }

// Resolve 保留旧传输接口，认证入口不能直接调用此无权限检查的解析器。
func (deferredTestStream) Resolve(string, int64) (string, error) { return "unsafe-fallback", nil }

// TestDeferredAuthorization 验证初始化未完成时拒绝请求，绑定后仍执行真实资源授权。
func TestDeferredAuthorization(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(deferredTestStream{})
	if err != nil {
		t.Fatal(err)
	}
	identity := &data.UserTokenPayload{UserId: 1, TenantId: 1, RoleCode: "operator"}
	_, _, err = registry.ResolveContext(t.Context(), "deferred", "tenant-two:project-seven", identity)
	if err == nil {
		t.Fatal("未绑定授权器时回落到旧解析器")
	}
	err = registry.SetContextResolver("deferred", authorizedTestStream{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = registry.ResolveContext(t.Context(), "deferred", "tenant-two:project-seven", identity)
	if err == nil {
		t.Fatal("绑定后普通用户越权成功")
	}
	identity.RoleCode = "admin"
	var resolved string
	resolved, _, err = registry.ResolveContext(t.Context(), "deferred", "tenant-two:project-seven", identity)
	if err != nil || resolved != "opaque-resource-key/tenant-two:project-seven" {
		t.Fatalf("绑定后授权请求不可用: %s, %v", resolved, err)
	}
}
