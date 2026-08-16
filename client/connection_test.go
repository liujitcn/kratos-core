package client

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNewConnectionWithMiddleware(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "client.Test",
		Methods: []grpc.MethodDesc{{
			MethodName: "Echo",
			Handler:    testUnaryHandler,
		}},
	}, nil)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	var calls atomic.Int32
	customMiddleware := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			calls.Add(1)
			return next(ctx, req)
		}
	}
	clientConfig := &configv1.Client{Grpc: &configv1.Client_Grpc{Endpoint: listener.Addr().String()}}
	var connection *Connection
	var cleanup func()
	connection, cleanup, err = NewConnection(context.Background(), clientConfig, WithMiddleware(customMiddleware))
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	t.Cleanup(cleanup)

	err = connection.Invoke(context.Background(), "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("middleware calls = %d, want 1", calls.Load())
	}
}

func TestNewLocalConnectionWithMiddleware(t *testing.T) {
	var calls atomic.Int32
	customMiddleware := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			calls.Add(1)
			return next(ctx, req)
		}
	}
	connection, cleanup, err := NewConnection(context.Background(), nil,
		WithMiddleware(customMiddleware),
		WithLocalServices(func(registrar grpc.ServiceRegistrar) {
			registrar.RegisterService(&grpc.ServiceDesc{
				ServiceName: "client.Test",
				Methods: []grpc.MethodDesc{{
					MethodName: "Echo",
					Handler:    testUnaryHandler,
				}},
			}, new(struct{}))
		}),
	)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	defer cleanup()

	err = connection.Invoke(context.Background(), "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("middleware calls = %d, want 1", calls.Load())
	}
}

func TestNewLocalConnectionWithConfiguredMiddleware(t *testing.T) {
	clientConfig := &configv1.Client{Grpc: &configv1.Client_Grpc{Middleware: &configv1.Client_Middleware{}}}
	connection, cleanup, err := NewConnection(context.Background(), clientConfig,
		WithLocalServices(func(registrar grpc.ServiceRegistrar) {
			registrar.RegisterService(&grpc.ServiceDesc{
				ServiceName: "client.Test",
				Methods: []grpc.MethodDesc{{
					MethodName: "Echo",
					Handler:    testMetadataUnaryHandler,
				}},
			}, new(struct{}))
		}),
	)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	defer cleanup()

	err = connection.Invoke(context.Background(), "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
}

func TestNewLocalConnectionPropagatesOutgoingMetadata(t *testing.T) {
	connection, cleanup, err := NewConnection(context.Background(), nil,
		WithLocalServices(func(registrar grpc.ServiceRegistrar) {
			registrar.RegisterService(&grpc.ServiceDesc{
				ServiceName: "client.Test",
				Methods: []grpc.MethodDesc{{
					MethodName: "Echo",
					Handler:    testMetadataUnaryHandler,
				}},
			}, new(struct{}))
		}),
	)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-request-id", "provided"))
	err = connection.Invoke(ctx, "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
}

func TestNewLocalConnectionPropagatesMiddlewareOutgoingMetadata(t *testing.T) {
	appendMetadata := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", "middleware")
			return next(ctx, req)
		}
	}
	connection, cleanup, err := NewConnection(context.Background(), nil,
		WithMiddleware(appendMetadata),
		WithLocalServices(func(registrar grpc.ServiceRegistrar) {
			registrar.RegisterService(&grpc.ServiceDesc{
				ServiceName: "client.Test",
				Methods: []grpc.MethodDesc{{
					MethodName: "Echo",
					Handler:    testMetadataUnaryHandler,
				}},
			}, new(struct{}))
		}),
	)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	defer cleanup()

	err = connection.Invoke(context.Background(), "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
}

func TestNewConnectionWithConfiguredMiddleware(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "client.Test",
		Methods: []grpc.MethodDesc{{
			MethodName: "Echo",
			Handler:    testMetadataUnaryHandler,
		}},
	}, nil)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	clientConfig := &configv1.Client{Grpc: &configv1.Client_Grpc{
		Endpoint:   listener.Addr().String(),
		Middleware: &configv1.Client_Middleware{},
	}}
	var connection *Connection
	var cleanup func()
	connection, cleanup, err = NewConnection(context.Background(), clientConfig)
	if err != nil {
		t.Fatalf("NewConnection() error = %v", err)
	}
	t.Cleanup(cleanup)

	err = connection.Invoke(context.Background(), "/client.Test/Echo", new(emptypb.Empty), new(emptypb.Empty))
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
}

func testUnaryHandler(service any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	if interceptor == nil {
		return testUnaryMethod(service, ctx, dec)
	}
	return interceptor(ctx, new(emptypb.Empty), &grpc.UnaryServerInfo{Server: service, FullMethod: "/client.Test/Echo"}, func(ctx context.Context, request any) (any, error) {
		return testUnaryMethod(service, ctx, func(message any) error {
			return dec(message)
		})
	})
}

func testUnaryMethod(_ any, _ context.Context, dec func(any) error) (any, error) {
	request := new(emptypb.Empty)
	if err := dec(request); err != nil {
		return nil, err
	}
	return new(emptypb.Empty), nil
}

func testMetadataUnaryHandler(service any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	method := func(ctx context.Context, request any) (any, error) {
		if err := dec(request); err != nil {
			return nil, err
		}
		if len(metadata.ValueFromIncomingContext(ctx, "x-request-id")) == 0 {
			return nil, status.Error(codes.InvalidArgument, "request id is missing")
		}
		return new(emptypb.Empty), nil
	}
	if interceptor == nil {
		return method(ctx, new(emptypb.Empty))
	}
	return interceptor(ctx, new(emptypb.Empty), &grpc.UnaryServerInfo{Server: service, FullMethod: "/client.Test/Echo"}, method)
}
