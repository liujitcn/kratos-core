package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/registry"
	"github.com/go-kratos/kratos/v3/selector"
	"github.com/go-kratos/kratos/v3/selector/filter"
	"github.com/go-kratos/kratos/v3/selector/p2c"
	"github.com/go-kratos/kratos/v3/selector/random"
	"github.com/go-kratos/kratos/v3/selector/wrr"
	"github.com/go-kratos/kratos/v3/transport"
	kratosGrpc "github.com/go-kratos/kratos/v3/transport/grpc"

	// HTTP transport initializes the shared selector before gRPC registers its balancer.
	_ "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/liujitcn/kratos-core/client/middleware/requestid"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const defaultTimeout = 5 * time.Second

// CreateGrpcClient 根据服务名或配置地址创建 gRPC 客户端。
func CreateGrpcClient(ctx context.Context, discovery registry.Discovery, serviceName string, config *configv1.Bootstrap, middlewares ...middleware.Middleware) (grpc.ClientConnInterface, error) {
	return createGrpcClient(ctx, discovery, serviceName, config, nil, nil, middlewares...)
}

// createGrpcClient 创建支持原生客户端拦截器的 gRPC 客户端连接。
func createGrpcClient(ctx context.Context, discovery registry.Discovery, serviceName string, config *configv1.Bootstrap, unaryInterceptors []grpc.UnaryClientInterceptor, streamInterceptors []grpc.StreamClientInterceptor, middlewares ...middleware.Middleware) (grpc.ClientConnInterface, error) {
	endpoint, err := resolveGrpcClientEndpoint(serviceName, config)
	if err != nil {
		return nil, err
	}

	options := make([]kratosGrpc.ClientOption, 0, 4)
	if strings.HasPrefix(endpoint, "discovery:///") {
		if discovery == nil {
			return nil, fmt.Errorf("gRPC 客户端使用服务发现地址时 Discovery 不能为空: %s", endpoint)
		}
		options = append(options, kratosGrpc.WithDiscovery(discovery))
	}
	options = append(options, kratosGrpc.WithEndpoint(endpoint))

	options, err = initGrpcClientConfigWithInterceptors(config, options, unaryInterceptors, streamInterceptors, middlewares...)
	if err != nil {
		return nil, fmt.Errorf("init grpc client config failed: %w", err)
	}

	var conn grpc.ClientConnInterface
	conn, err = kratosGrpc.NewClient(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("dial grpc client [%s] failed: %w", endpoint, err)
	}
	return conn, nil
}

// resolveGrpcClientEndpoint 按显式服务名优先、配置地址回退的顺序解析客户端地址。
func resolveGrpcClientEndpoint(serviceName string, config *configv1.Bootstrap) (string, error) {
	if serviceName != "" {
		if strings.Contains(serviceName, "://") {
			return serviceName, nil
		}
		return "discovery:///" + serviceName, nil
	}
	if config == nil || config.GetClient() == nil || config.GetClient().GetGrpc() == nil {
		return "", fmt.Errorf("gRPC 客户端服务名和配置地址不能同时为空")
	}

	endpoint := config.GetClient().GetGrpc().GetEndpoint()
	if endpoint == "" {
		return "", fmt.Errorf("gRPC 客户端服务名和配置地址不能同时为空")
	}
	if strings.Contains(endpoint, "://") {
		return endpoint, nil
	}
	return "direct:///" + endpoint, nil
}

// initGrpcClientConfig 根据客户端配置组装 gRPC 客户端选项。
func initGrpcClientConfig(config *configv1.Bootstrap, options []kratosGrpc.ClientOption, middlewares ...middleware.Middleware) ([]kratosGrpc.ClientOption, error) {
	return initGrpcClientConfigWithInterceptors(config, options, nil, nil, middlewares...)
}

// initGrpcClientConfigWithInterceptors 根据配置组装 Kratos middleware 和原生 gRPC 客户端拦截器。
func initGrpcClientConfigWithInterceptors(config *configv1.Bootstrap, options []kratosGrpc.ClientOption, unaryInterceptors []grpc.UnaryClientInterceptor, streamInterceptors []grpc.StreamClientInterceptor, middlewares ...middleware.Middleware) ([]kratosGrpc.ClientOption, error) {
	if config == nil || config.GetClient() == nil || config.GetClient().GetGrpc() == nil {
		err := configureGlobalSelector("wrr")
		if err != nil {
			return nil, err
		}
		if len(middlewares) > 0 {
			options = append(options, kratosGrpc.WithMiddleware(middlewares...))
		}
		options = appendClientInterceptors(options, middlewares, configuredClientInterceptors{}, unaryInterceptors, streamInterceptors)
		return options, nil
	}

	grpcConfig := config.GetClient().GetGrpc()
	timeout := defaultTimeout
	if grpcConfig.Timeout != nil {
		timeout = grpcConfig.Timeout.AsDuration()
	}
	options = append(options, kratosGrpc.WithTimeout(timeout))

	var err error
	middlewares, err = appendClientMiddlewaresWithMetadata(grpcConfig.Middleware, grpcConfig.Metadata, middlewares...)
	if err != nil {
		return nil, err
	}
	middlewareConfig := grpcConfig.GetMiddleware()
	var configuredInterceptors configuredClientInterceptors
	configuredInterceptors, err = buildConfiguredClientInterceptors(middlewareConfig)
	if err != nil {
		return nil, err
	}
	balancerName := "wrr"
	if middlewareConfig != nil {
		selectorFilterConfig := middlewareConfig.GetSelectorFilter()
		if selectorFilterConfig != nil {
			if selectorFilterConfig.GetFilterVersion() != "" {
				options = append(options, kratosGrpc.WithNodeFilter(filter.Version(selectorFilterConfig.GetFilterVersion())))
			}
			switch selectorFilterConfig.GetBalancer() {
			case "p2c":
				balancerName = "p2c"
			case "random":
				balancerName = "random"
			case "wrr":
				balancerName = "wrr"
			default:
				balancerName = "wrr"
			}
		}
	}
	err = configureGlobalSelector(balancerName)
	if err != nil {
		return nil, err
	}
	options = append(options, kratosGrpc.WithMiddleware(middlewares...))
	options = appendClientInterceptors(options, middlewares, configuredInterceptors, unaryInterceptors, streamInterceptors)

	if grpcConfig.Tls == nil {
		return options, nil
	}
	var tlsConfig *tls.Config
	tlsConfig, err = utils.LoadClientTlsConfig(grpcConfig.Tls)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		options = append(options, kratosGrpc.WithTLSConfig(tlsConfig))
	}
	return options, nil
}

// appendClientInterceptors 按请求 ID、配置拦截器、自定义拦截器和流式 middleware 的顺序追加客户端拦截器。
func appendClientInterceptors(options []kratosGrpc.ClientOption, middlewares []middleware.Middleware, configured configuredClientInterceptors, unaryInterceptors []grpc.UnaryClientInterceptor, streamInterceptors []grpc.StreamClientInterceptor) []kratosGrpc.ClientOption {
	unary := clientUnaryInterceptors(configured, unaryInterceptors)
	streams := clientStreamInterceptors(configured, streamInterceptors)
	streams = append(streams, streamMiddlewareInterceptor(middlewares...))
	streams = append(streams, requestid.StreamClientInterceptor())
	return append(options,
		kratosGrpc.WithUnaryInterceptor(unary...),
		kratosGrpc.WithStreamInterceptor(streams...),
	)
}

// clientUnaryInterceptors 组装请求 ID、配置项和调用方 unary 拦截器链。
func clientUnaryInterceptors(configured configuredClientInterceptors, custom []grpc.UnaryClientInterceptor) []grpc.UnaryClientInterceptor {
	interceptors := []grpc.UnaryClientInterceptor{requestid.UnaryClientInterceptor()}
	interceptors = append(interceptors, configured.unary...)
	return append(interceptors, custom...)
}

// clientStreamInterceptors 组装配置项和调用方 stream 拦截器链。
func clientStreamInterceptors(configured configuredClientInterceptors, custom []grpc.StreamClientInterceptor) []grpc.StreamClientInterceptor {
	interceptors := make([]grpc.StreamClientInterceptor, 0, len(configured.stream)+len(custom))
	interceptors = append(interceptors, configured.stream...)
	return append(interceptors, custom...)
}

// streamMiddlewareInterceptor 在创建 gRPC 流前执行客户端 middleware，并传播 transport 请求头。
func streamMiddlewareInterceptor(middlewares ...middleware.Middleware) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, callOptions ...grpc.CallOption) (grpc.ClientStream, error) {
		handler := func(nextCtx context.Context, _ any) (any, error) {
			if clientTransport, ok := transport.FromClientContext(nextCtx); ok {
				keys := clientTransport.RequestHeader().Keys()
				values := make([]string, 0, len(keys)*2)
				for _, key := range keys {
					values = append(values, key, clientTransport.RequestHeader().Get(key))
				}
				if len(values) > 0 {
					nextCtx = metadata.AppendToOutgoingContext(nextCtx, values...)
				}
			}
			return streamer(nextCtx, desc, cc, method, callOptions...)
		}
		if len(middlewares) > 0 {
			handler = middleware.Chain(middlewares...)(handler)
		}
		result, err := handler(ctx, nil)
		if err != nil {
			return nil, err
		}
		stream, ok := result.(grpc.ClientStream)
		if !ok {
			return nil, fmt.Errorf("gRPC stream middleware returned %T", result)
		}
		return stream, nil
	}
}

var globalSelectorState struct {
	sync.Mutex
	configured bool
	name       string
}

// configureGlobalSelector 配置进程级 gRPC 负载均衡器，并拒绝运行期间切换策略。
func configureGlobalSelector(name string) error {
	globalSelectorState.Lock()
	defer globalSelectorState.Unlock()

	if globalSelectorState.configured {
		if globalSelectorState.name != name {
			return fmt.Errorf("gRPC 客户端负载均衡器已配置为 %q，不能切换为 %q", globalSelectorState.name, name)
		}
		return nil
	}

	switch name {
	case "p2c":
		selector.SetGlobalSelector(p2c.NewBuilder())
	case "random":
		selector.SetGlobalSelector(random.NewBuilder())
	default:
		selector.SetGlobalSelector(wrr.NewBuilder())
		name = "wrr"
	}
	globalSelectorState.configured = true
	globalSelectorState.name = name
	return nil
}
