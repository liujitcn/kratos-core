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
	selectorFilter "github.com/go-kratos/kratos/v3/selector/filter"
	selectorP2c "github.com/go-kratos/kratos/v3/selector/p2c"
	selectorRandom "github.com/go-kratos/kratos/v3/selector/random"
	selectorWrr "github.com/go-kratos/kratos/v3/selector/wrr"
	kratosGrpc "github.com/go-kratos/kratos/v3/transport/grpc"

	// HTTP transport initializes the shared selector before gRPC registers its balancer.
	_ "github.com/go-kratos/kratos/v3/transport/http"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/utils"
	"google.golang.org/grpc"
)

const defaultTimeout = 5 * time.Second

// CreateGrpcClient 根据服务名或配置地址创建 gRPC 客户端。
func CreateGrpcClient(ctx context.Context, discovery registry.Discovery, serviceName string, config *configv1.Bootstrap, middlewares ...middleware.Middleware) (grpc.ClientConnInterface, error) {
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

	options, err = initGrpcClientConfig(config, options, middlewares...)
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
	if config == nil || config.GetClient() == nil || config.GetClient().GetGrpc() == nil {
		err := configureGlobalSelector("wrr")
		if err != nil {
			return nil, err
		}
		if len(middlewares) > 0 {
			options = append(options,
				kratosGrpc.WithMiddleware(middlewares...),
				kratosGrpc.WithStreamMiddleware(middlewares...),
			)
		}
		return options, nil
	}

	grpcConfig := config.GetClient().GetGrpc()
	timeout := defaultTimeout
	if grpcConfig.Timeout != nil {
		timeout = grpcConfig.Timeout.AsDuration()
	}
	options = append(options, kratosGrpc.WithTimeout(timeout))

	var err error
	middlewares, err = appendClientMiddlewares(grpcConfig.Middleware, middlewares)
	if err != nil {
		return nil, err
	}
	middlewareConfig := grpcConfig.GetMiddleware()
	balancerName := "wrr"
	if middlewareConfig != nil {
		selectorFilterConfig := middlewareConfig.GetSelectorFilter()
		if selectorFilterConfig != nil {
			if selectorFilterConfig.GetFilterVersion() != "" {
				options = append(options, kratosGrpc.WithNodeFilter(selectorFilter.Version(selectorFilterConfig.GetFilterVersion())))
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
	options = append(options,
		kratosGrpc.WithMiddleware(middlewares...),
		kratosGrpc.WithStreamMiddleware(middlewares...),
	)

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
		selector.SetGlobalSelector(selectorP2c.NewBuilder())
	case "random":
		selector.SetGlobalSelector(selectorRandom.NewBuilder())
	default:
		selector.SetGlobalSelector(selectorWrr.NewBuilder())
		name = "wrr"
	}
	globalSelectorState.configured = true
	globalSelectorState.name = name
	return nil
}
