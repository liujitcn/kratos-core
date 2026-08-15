package sse

import (
	"context"
	"errors"
	"net/http"
	"sync"
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
