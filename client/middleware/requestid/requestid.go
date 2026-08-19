// Package requestid 提供跨 HTTP 和 gRPC 的请求标识生成与透传中间件。
package requestid

import (
	"context"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/go-utils/id"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// DefaultRequestIDHeader 是默认的请求标识请求头。
const DefaultRequestIDHeader = "X-Request-Id"

// RequestIDOption 配置请求标识中间件行为。
type RequestIDOption func(*requestIDOptions)

type requestIDOptions struct {
	headerName string
	generator  func() string
}

// WithRequestIDHeader 设置自定义请求标识头名称。
func WithRequestIDHeader(name string) RequestIDOption {
	return func(options *requestIDOptions) {
		if name != "" {
			options.headerName = name
		}
	}
}

// WithRequestIDGenerator 设置自定义请求标识生成函数。
func WithRequestIDGenerator(generator func() string) RequestIDOption {
	return func(options *requestIDOptions) {
		if generator != nil {
			options.generator = generator
		}
	}
}

type requestIDKey struct{}

// WithRequestID 将请求标识写入上下文。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// FromContext 从上下文读取请求标识，不存在时返回空字符串。
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// GetRequestID 从上下文读取请求标识。
func GetRequestID(ctx context.Context) string {
	return FromContext(ctx)
}

// Server 创建服务端请求标识中间件。
func Server(opts ...RequestIDOption) middleware.Middleware {
	options := newOptions(opts)
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			requestID := FromContext(ctx)
			serverTransport, ok := transport.FromServerContext(ctx)
			if requestID == "" && ok {
				requestID = serverTransport.RequestHeader().Get(options.headerName)
			}
			if requestID == "" {
				requestID = options.generator()
			}
			if ok {
				serverTransport.ReplyHeader().Set(options.headerName, requestID)
			}
			return next(WithRequestID(ctx, requestID), req)
		}
	}
}

// Client 创建客户端请求标识中间件。
func Client(opts ...RequestIDOption) middleware.Middleware {
	options := newOptions(opts)
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			requestID := FromContext(ctx)
			if requestID == "" {
				requestID = options.generator()
				ctx = WithRequestID(ctx, requestID)
			}
			clientTransport, ok := transport.FromClientContext(ctx)
			if ok {
				clientTransport.RequestHeader().Set(options.headerName, requestID)
			}
			return next(ctx, req)
		}
	}
}

// UnaryClientInterceptor 创建将上下文请求标识写入 outgoing metadata 的一元客户端拦截器。
func UnaryClientInterceptor(opts ...RequestIDOption) grpc.UnaryClientInterceptor {
	options := newOptions(opts)
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, callOpts ...grpc.CallOption) error {
		requestID := FromContext(ctx)
		if requestID == "" {
			requestID = options.generator()
			ctx = WithRequestID(ctx, requestID)
		}
		if outgoing, ok := metadata.FromOutgoingContext(ctx); ok && len(outgoing.Get(options.headerName)) > 0 {
			return invoker(ctx, method, req, reply, cc, callOpts...)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, options.headerName, requestID)
		return invoker(ctx, method, req, reply, cc, callOpts...)
	}
}

// StreamClientInterceptor 创建将上下文请求标识写入 outgoing metadata 的流式客户端拦截器。
func StreamClientInterceptor(opts ...RequestIDOption) grpc.StreamClientInterceptor {
	options := newOptions(opts)
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOpts ...grpc.CallOption) (grpc.ClientStream, error) {
		requestID := FromContext(ctx)
		if requestID == "" {
			requestID = options.generator()
			ctx = WithRequestID(ctx, requestID)
		}
		if outgoing, ok := metadata.FromOutgoingContext(ctx); ok && len(outgoing.Get(options.headerName)) > 0 {
			return streamer(ctx, desc, cc, method, callOpts...)
		}
		ctx = metadata.AppendToOutgoingContext(ctx, options.headerName, requestID)
		return streamer(ctx, desc, cc, method, callOpts...)
	}
}

// NewRequestIDMiddleware 创建兼容旧调用方式的服务端请求标识中间件。
func NewRequestIDMiddleware(opts ...RequestIDOption) middleware.Middleware {
	return Server(opts...)
}

// newOptions 创建请求标识中间件的默认配置。
func newOptions(opts []RequestIDOption) *requestIDOptions {
	options := &requestIDOptions{
		headerName: DefaultRequestIDHeader,
		generator:  id.NewGUIDv7,
	}
	for _, option := range opts {
		if option != nil {
			option(options)
		}
	}
	return options
}
