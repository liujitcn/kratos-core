package sse

import (
	"fmt"
	"sync"

	"github.com/liujitcn/kratos-kit/transport/sse"
)

// Registry 保存已注册的 SSE 业务流定义，并为 Core 请求层提供解析能力。
type Registry struct {
	mu      sync.RWMutex
	streams map[string]sse.SSEStream
}

// tenantResolver 支持从认证主体中获取租户隔离信息的 SSE 流。
type tenantResolver interface {
	ResolveTenant(channelID string, userID, tenantID int64) (string, error)
}

// NewRegistry 创建空 SSE 流注册表。
func NewRegistry() *Registry {
	return &Registry{streams: make(map[string]sse.SSEStream)}
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

// resolve 兼容旧流并优先调用支持租户隔离的流实现。
func (r *Registry) resolve(streamID, channelID string, userID, tenantID int64) (string, bool, error) {
	r.mu.RLock()
	stream, exists := r.streams[streamID]
	r.mu.RUnlock()
	if !exists {
		return "", false, nil
	}
	var transportID string
	var err error
	if tenantID > 0 {
		if tenantStream, ok := stream.(tenantResolver); ok {
			transportID, err = tenantStream.ResolveTenant(channelID, userID, tenantID)
		} else {
			transportID, err = stream.Resolve(channelID, userID)
		}
	} else {
		transportID, err = stream.Resolve(channelID, userID)
	}
	if err != nil {
		return "", true, err
	}
	return transportID, true, nil
}
