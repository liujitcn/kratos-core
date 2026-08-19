// Package ratelimit 提供 gRPC 客户端请求限流拦截器。
package ratelimit

import (
	"context"

	coreRateLimit "github.com/liujitcn/kratos-kit/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Option 配置客户端限流拦截器。
type Option func(*options)

type options struct {
	waitMode    bool
	skipMethods map[string]bool
}

// WithWait 启用等待模式，请求等待限流器放行或上下文取消。
func WithWait() Option {
	return func(options *options) {
		options.waitMode = true
	}
}

// WithSkipMethods 添加跳过限流的完整 gRPC 方法名。
func WithSkipMethods(methods ...string) Option {
	return func(options *options) {
		if options.skipMethods == nil {
			options.skipMethods = make(map[string]bool)
		}
		for _, method := range methods {
			options.skipMethods[method] = true
		}
	}
}

// UnaryClientInterceptor 创建限制出站一元 RPC 速率的客户端拦截器。
func UnaryClientInterceptor(limiter coreRateLimit.Limiter, interceptorOptions ...Option) grpc.UnaryClientInterceptor {
	config := &options{}
	for _, option := range interceptorOptions {
		option(config)
	}
	return func(ctx context.Context, method string, request any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		if config.skipMethods[method] {
			return invoker(ctx, method, request, reply, conn, callOptions...)
		}
		err := acquire(ctx, limiter, config.waitMode)
		if err != nil {
			return err
		}
		return invoker(ctx, method, request, reply, conn, callOptions...)
	}
}

// StreamClientInterceptor 创建限制出站流式 RPC 建流速率的客户端拦截器。
func StreamClientInterceptor(limiter coreRateLimit.Limiter, interceptorOptions ...Option) grpc.StreamClientInterceptor {
	config := &options{}
	for _, option := range interceptorOptions {
		option(config)
	}
	return func(ctx context.Context, description *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, callOptions ...grpc.CallOption) (grpc.ClientStream, error) {
		if config.skipMethods[method] {
			return streamer(ctx, description, conn, method, callOptions...)
		}
		err := acquire(ctx, limiter, config.waitMode)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, description, conn, method, callOptions...)
	}
}

// acquire 从限流器获取一次请求许可。
func acquire(ctx context.Context, limiter coreRateLimit.Limiter, waitMode bool) error {
	if waitMode {
		err := limiter.Wait(ctx)
		if err != nil {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return nil
	}
	allowed, err := limiter.Allow()
	if !allowed || err != nil {
		return status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return nil
}
