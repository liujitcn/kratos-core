// Package retry 提供 gRPC 客户端一元请求重试拦截器。
package retry

import (
	"context"
	"strings"

	"github.com/liujitcn/kratos-kit/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var defaultIdempotentPrefixes = []string{"Get", "List", "Search"}

var defaultRetryCodes = map[codes.Code]bool{
	codes.Unavailable: true,
}

// Option 配置客户端重试拦截器。
type Option func(*options)

type options struct {
	idempotentPrefixes []string
	retryCodes         map[codes.Code]bool
	skipMethods        map[string]bool
}

// WithIdempotentPrefixes 设置用于识别幂等 RPC 的方法名前缀。
func WithIdempotentPrefixes(prefixes ...string) Option {
	return func(options *options) {
		options.idempotentPrefixes = prefixes
	}
}

// WithRetryCodes 设置触发重试的 gRPC 状态码。
func WithRetryCodes(retryCodes ...codes.Code) Option {
	return func(options *options) {
		options.retryCodes = make(map[codes.Code]bool, len(retryCodes))
		for _, code := range retryCodes {
			options.retryCodes[code] = true
		}
	}
}

// WithSkipMethods 添加跳过重试的完整 gRPC 方法名。
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

// UnaryClientInterceptor 创建在服务端返回临时错误时重试幂等出站 RPC 的客户端拦截器。
func UnaryClientInterceptor(retrier *retry.Retrier, interceptorOptions ...Option) grpc.UnaryClientInterceptor {
	config := &options{
		idempotentPrefixes: defaultIdempotentPrefixes,
		retryCodes:         defaultRetryCodes,
	}
	for _, option := range interceptorOptions {
		option(config)
	}

	return func(ctx context.Context, method string, request any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		if config.skipMethods[method] || !isIdempotent(method, config.idempotentPrefixes) {
			return invoker(ctx, method, request, reply, conn, callOptions...)
		}

		var lastErr error
		retryErr := retrier.Do(ctx, func(attemptCtx context.Context) error {
			var err error
			err = invoker(attemptCtx, method, request, reply, conn, callOptions...)
			lastErr = err
			if err == nil {
				return nil
			}
			if config.retryCodes[status.Code(err)] {
				return err
			}
			return nil
		})
		if lastErr == nil {
			return retryErr
		}
		return lastErr
	}
}

// isIdempotent 检查 gRPC 完整方法名是否匹配任一幂等方法前缀。
func isIdempotent(fullMethod string, prefixes []string) bool {
	methodName := fullMethod
	if _, method, found := strings.CutLast(fullMethod, "/"); found {
		methodName = method
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(methodName, prefix) {
			return true
		}
	}
	return false
}
