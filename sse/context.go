package sse

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
)

// DetachRequestContext 为长连接请求移除服务级 deadline，同时保留客户端断开带来的取消信号。
func DetachRequestContext(request *http.Request) (*http.Request, func()) {
	if request == nil || request.Context() == nil {
		return request, func() {}
	}
	originalContext := request.Context()
	streamContext, cancel := context.WithCancel(context.WithoutCancel(originalContext))
	finished := make(chan struct{})
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			close(finished)
			cancel()
		})
	}
	go func() {
		select {
		case <-originalContext.Done():
			if !errors.Is(originalContext.Err(), context.DeadlineExceeded) {
				cancel()
			}
		case <-finished:
		}
	}()
	return request.WithContext(streamContext), cleanup
}

// watchSubscription 在长连接期间复查会话及资源授权，退出或授权撤销后停止发送和回放。
func watchSubscription(request *http.Request, registry *Registry, authenticator engine.Authenticator, userToken *data.UserToken, streamID, channelID, transportID string) (*http.Request, context.CancelFunc) {
	ctx, cancel := context.WithCancel(request.Context())
	snapshot := request.Clone(ctx)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				identity, err := requestIdentity(snapshot, streamID, authenticator, userToken)
				if err != nil {
					cancel()
					return
				}
				checkContext, stop := context.WithTimeout(ctx, 5*time.Second)
				var current string
				var found bool
				current, found, err = registry.ResolveContext(checkContext, streamID, channelID, identity)
				stop()
				if err != nil || !found || current != transportID {
					cancel()
					return
				}
			}
		}
	}()
	return snapshot, cancel
}
