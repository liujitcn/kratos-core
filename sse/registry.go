package sse

import (
	"context"
	"fmt"
	"sync"

	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/transport/sse"

	"github.com/liujitcn/kratos-core/errorsx"
)

// Registry 保存已注册的 SSE 业务流定义，并为 Core 请求层提供解析能力。
type Registry struct {
	mu        sync.RWMutex
	streams   map[string]sse.SSEStream
	resolvers map[string]ContextResolver
}

// tenantResolver 支持从认证主体中获取租户隔离信息的 SSE 流。
type tenantResolver interface {
	ResolveTenant(channelID string, userID, tenantID int64) (string, error)
}

// ContextResolver 使用已验证的身份和上下文检查流的资源及操作权限。
type ContextResolver interface {
	ResolveContext(context.Context, string, *data.UserTokenPayload) (string, error)
}

// NewRegistry 创建空 SSE 流注册表。
func NewRegistry() *Registry {
	return &Registry{streams: make(map[string]sse.SSEStream), resolvers: make(map[string]ContextResolver)}
}

// SetContextResolver 在业务客户端就绪后绑定授权器，避免 SSE 创建与业务模块之间形成依赖环。
func (r *Registry) SetContextResolver(streamID string, resolver ContextResolver) error {
	if streamID == "" || resolver == nil {
		return fmt.Errorf("SSE授权器及流标识不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resolvers[streamID] = resolver
	return nil
}

// Register 注册 SSE 流定义，并拒绝空对象、空标识和重复标识。
func (r *Registry) Register(streams ...sse.SSEStream) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	registered := make(map[string]struct{}, len(streams))
	for _, stream := range streams {
		if stream == nil {
			return fmt.Errorf("SSE流不能为空")
		}
		streamID := stream.ID()
		if streamID == "" {
			return fmt.Errorf("SSE流标识不能为空")
		}
		if _, exists := r.streams[streamID]; exists {
			return fmt.Errorf("SSE流标识重复: %s", streamID)
		}
		if _, exists := registered[streamID]; exists {
			return fmt.Errorf("SSE流标识重复: %s", streamID)
		}
		registered[streamID] = struct{}{}
	}
	for _, stream := range streams {
		r.streams[stream.ID()] = stream
	}
	return nil
}

// Resolve 解析订阅请求对应的传输流标识。
func (r *Registry) Resolve(streamID, channelID string, userID int64) (string, bool, error) {
	return r.resolve(streamID, channelID, userID, 0)
}

// ResolveWithTenant 解析带可信租户上下文的 SSE 流。
func (r *Registry) ResolveWithTenant(streamID, channelID string, userID, tenantID int64) (string, bool, error) {
	return r.resolve(streamID, channelID, userID, tenantID)
}

// ResolveContext 在回放或发送任何事件前执行流声明的授权，不推断频道字符串的资源语义。
func (r *Registry) ResolveContext(ctx context.Context, streamID, channelID string, identity *data.UserTokenPayload) (string, bool, error) {
	r.mu.RLock()
	stream, exists := r.streams[streamID]
	boundResolver := r.resolvers[streamID]
	r.mu.RUnlock()
	if !exists {
		return "", false, nil
	}
	if identity == nil || identity.UserId <= 0 {
		return "", true, errorsx.Unauthenticated("SSE访问主体无效")
	}
	ctx = engine.ContextWithAuthClaims(ctx, identity.MakeAuthClaims())
	var transportID string
	var err error
	if boundResolver != nil {
		transportID, err = boundResolver.ResolveContext(ctx, channelID, identity)
	} else if resolver, ok := stream.(ContextResolver); ok {
		transportID, err = resolver.ResolveContext(ctx, channelID, identity)
	} else if resolver, ok := stream.(tenantResolver); ok && identity.TenantId > 0 {
		transportID, err = resolver.ResolveTenant(channelID, identity.UserId, identity.TenantId)
	} else {
		return "", true, errorsx.PermissionDenied("SSE流未声明资源授权边界")
	}
	if err != nil {
		return "", true, err
	}
	if transportID == "" {
		return "", true, errorsx.PermissionDenied("SSE流授权结果为空")
	}
	return transportID, true, nil
}

// resolve 使用受信身份执行统一流授权，缺少租户或角色信息不会被推断为平台权限。
func (r *Registry) resolve(streamID, channelID string, userID, tenantID int64) (string, bool, error) {
	return r.ResolveContext(context.Background(), streamID, channelID, &data.UserTokenPayload{UserId: userID, TenantId: tenantID})
}
