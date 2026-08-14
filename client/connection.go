package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	kratosRegistry "github.com/go-kratos/kratos/v3/registry"
	"github.com/liujitcn/kratos-core/client/localgrpc"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	kitRPC "github.com/liujitcn/kratos-kit/rpc"
	"google.golang.org/grpc"
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
	discovery     kratosRegistry.Discovery
	localServices []LocalServiceRegistrar
}

// WithDiscovery 为使用 discovery:/// 地址的客户端连接注入服务发现器。
func WithDiscovery(discovery kratosRegistry.Discovery) Option {
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
		conn := localgrpc.NewConn()
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
	conn, err = kitRPC.CreateGrpcClient(ctx, connectionOpts.discovery, "", rpcConfig)
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
