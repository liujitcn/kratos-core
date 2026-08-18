package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/registry"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/client/localgrpc"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ grpc.ClientConnInterface = (*Connection)(nil)

// Connection 统一适配 Admin 的 gRPC 客户端连接。
type Connection struct {
	mu   sync.RWMutex
	conn grpc.ClientConnInterface
}

// Option 配置客户端连接的可选依赖。
type Option func(*connectionOptions)

// LocalServiceRegistrar 描述向进程内 gRPC 客户端注册服务的函数。
type LocalServiceRegistrar func(grpc.ServiceRegistrar)

type connectionOptions struct {
	discovery     registry.Discovery
	localServices []LocalServiceRegistrar
	middlewares   []middleware.Middleware
}

// WithDiscovery 为使用 discovery:/// 地址的客户端连接注入服务发现器。
func WithDiscovery(discovery registry.Discovery) Option {
	return func(options *connectionOptions) {
		options.discovery = discovery
	}
}

// WithLocalServices 配置进程内 gRPC 客户端需要注册的服务。
func WithLocalServices(registrars ...LocalServiceRegistrar) Option {
	return func(options *connectionOptions) {
		options.localServices = append(options.localServices, registrars...)
	}
}

// WithMiddleware 为 gRPC 客户端连接追加 Kratos 客户端中间件。
func WithMiddleware(middlewares ...middleware.Middleware) Option {
	return func(options *connectionOptions) {
		options.middlewares = append(options.middlewares, middlewares...)
	}
}

// NewConnection 根据客户端配置初始化 gRPC 客户端连接。
// endpoint 使用普通地址时直连本地或远程服务，使用 discovery:/// 前缀时通过 WithDiscovery 注入的注册中心发现服务；客户端配置为空时创建进程内连接。
func NewConnection(ctx context.Context, clientConfig *configv1.Client, options ...Option) (*Connection, func(), error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("客户端上下文不能为空")
	}

	connectionOpts := &connectionOptions{}
	for _, option := range options {
		if option != nil {
			option(connectionOpts)
		}
	}
	if clientConfig == nil || clientConfig.GetGrpc() == nil || clientConfig.GetGrpc().GetEndpoint() == "" {
		localMiddlewares := connectionOpts.middlewares
		var err error
		if clientConfig != nil && clientConfig.GetGrpc() != nil {
			localMiddlewares, err = appendClientMiddlewares(clientConfig.GetGrpc().GetMiddleware(), localMiddlewares)
			if err != nil {
				return nil, nil, fmt.Errorf("创建本地客户端中间件: %w", err)
			}
		}
		localOptions := []localgrpc.Option{
			localgrpc.WithUnaryInterceptor(newLocalUnaryInterceptor(localMiddlewares)),
			localgrpc.WithStreamInterceptor(newLocalStreamInterceptor(localMiddlewares)),
		}
		conn := localgrpc.NewConn(localOptions...)
		for _, register := range connectionOpts.localServices {
			if register != nil {
				register(conn)
			}
		}
		connection := &Connection{conn: conn}
		return connection, func() {}, nil
	}

	rpcConfig := &configv1.Bootstrap{Client: clientConfig}
	var conn grpc.ClientConnInterface
	var err error
	conn, err = CreateGrpcClient(ctx, connectionOpts.discovery, "", rpcConfig, connectionOpts.middlewares...)
	if err != nil {
		return nil, nil, fmt.Errorf("创建客户端 gRPC 连接: %w", err)
	}
	connection := &Connection{conn: conn}
	return connection, func() {
		closeConnection(conn)
	}, nil
}

// Invoke 将一元 RPC 调用转发给当前连接。
func (c *Connection) Invoke(ctx context.Context, method string, args any, reply any, options ...grpc.CallOption) error {
	conn, err := c.current()
	if err != nil {
		return err
	}
	return conn.Invoke(ctx, method, args, reply, options...)
}

// NewStream 将流式 RPC 调用转发给当前连接。
func (c *Connection) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, options ...grpc.CallOption) (grpc.ClientStream, error) {
	conn, err := c.current()
	if err != nil {
		return nil, err
	}
	return conn.NewStream(ctx, desc, method, options...)
}

// ClientConn 返回底层的 gRPC 客户端连接接口。
func (c *Connection) ClientConn() grpc.ClientConnInterface {
	conn, err := c.current()
	if err != nil {
		return nil
	}
	return conn
}

// current 返回已经初始化的底层连接。
func (c *Connection) current() (grpc.ClientConnInterface, error) {
	if c == nil {
		return nil, fmt.Errorf("客户端连接尚未初始化")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn == nil {
		return nil, fmt.Errorf("客户端连接尚未初始化")
	}
	return c.conn, nil
}

// closeConnection 关闭底层连接并记录关闭失败。
func closeConnection(conn grpc.ClientConnInterface) {
	closer, ok := conn.(interface{ Close() error })
	if !ok {
		return
	}
	err := closer.Close()
	if err != nil {
		log.Error("关闭客户端 gRPC 连接失败", "error", err)
	}
}

// newLocalUnaryInterceptor 将 Kratos middleware 适配为进程内 unary 拦截器。
func newLocalUnaryInterceptor(middlewares []middleware.Middleware) grpc.UnaryServerInterceptor {
	chain := middleware.Chain(middlewares...)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestHeader := make(metadata.MD)
		if outgoing, ok := metadata.FromOutgoingContext(ctx); ok {
			requestHeader = outgoing.Copy()
		}
		operation := ""
		if info != nil {
			operation = info.FullMethod
		}
		localTransport := newLocalTransport(operation, requestHeader)
		ctx = transport.NewClientContext(ctx, localTransport)
		response, err := chain(func(ctx context.Context, req any) (any, error) {
			serverContext := transport.NewServerContext(ctx, localTransport)
			serverContext = metadata.NewIncomingContext(serverContext, localIncomingMetadata(ctx, localTransport))
			serverContext = grpc.NewContextWithServerTransportStream(serverContext, localTransport)
			return handler(serverContext, req)
		})(ctx, req)
		if responseMetadata := localgrpc.ResponseMetadataFromContext(ctx); responseMetadata != nil {
			responseMetadata.Header = metadata.MD(localTransport.replyHeader).Copy()
			responseMetadata.Trailer = localTransport.replyTrailer.Copy()
		}
		return response, err
	}
}

// newLocalStreamInterceptor 将 Kratos middleware 适配为进程内 stream 拦截器。
func newLocalStreamInterceptor(middlewares []middleware.Middleware) grpc.StreamServerInterceptor {
	chain := middleware.Chain(middlewares...)
	return func(server any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		requestHeader := make(metadata.MD)
		if outgoing, ok := metadata.FromOutgoingContext(stream.Context()); ok {
			requestHeader = outgoing.Copy()
		}
		operation := ""
		if info != nil {
			operation = info.FullMethod
		}
		localTransport := newLocalTransport(operation, requestHeader)
		ctx := transport.NewClientContext(stream.Context(), localTransport)
		_, err := chain(func(ctx context.Context, _ any) (any, error) {
			serverContext := transport.NewServerContext(ctx, localTransport)
			serverContext = metadata.NewIncomingContext(serverContext, localIncomingMetadata(ctx, localTransport))
			serverContext = grpc.NewContextWithServerTransportStream(serverContext, localStreamTransport{stream: stream, operation: operation})
			if setter, ok := stream.(interface{ SetContext(context.Context) }); ok {
				setter.SetContext(serverContext)
			}
			return nil, handler(server, stream)
		})(ctx, nil)
		return err
	}
}

// localIncomingMetadata 合并 middleware 链结束后的 outgoing metadata 与 transport 请求头。
func localIncomingMetadata(ctx context.Context, localTransport *localTransport) metadata.MD {
	requestHeader := metadata.MD(localTransport.requestHeader).Copy()
	outgoing, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return requestHeader
	}
	for key := range localTransport.initialRequestHeader {
		delete(requestHeader, key)
	}
	for key, values := range outgoing {
		requestHeader[key] = append([]string(nil), values...)
	}
	return requestHeader
}
