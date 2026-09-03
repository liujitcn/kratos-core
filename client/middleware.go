package client

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/client/middleware/requestid"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine/jwt"
	authnMiddleware "github.com/liujitcn/kratos-kit/auth/authn/middleware"
	"github.com/liujitcn/kratos-kit/sdk"
	"github.com/liujitcn/kratos-kit/tracing"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const jwtKeyName = "kratos-kit:authn/jwt"

// appendClientMiddlewares 按客户端配置追加 Kratos 中间件。
func appendClientMiddlewares(config *configv1.Client_Middleware, middlewares []middleware.Middleware) ([]middleware.Middleware, error) {
	return appendClientMiddlewaresWithMetadata(config, nil, middlewares...)
}

// appendClientMiddlewaresWithMetadata 按传输层、业务层顺序组装客户端中间件。
func appendClientMiddlewaresWithMetadata(config *configv1.Client_Middleware, staticMetadata map[string]string, custom ...middleware.Middleware) ([]middleware.Middleware, error) {
	result := make([]middleware.Middleware, 0, len(custom)+8)
	if config == nil {
		if len(staticMetadata) > 0 {
			result = append(result, clientMetadata(staticMetadata))
		}
		return append(result, custom...), nil
	}
	result = append(result, requestid.Client())
	if config.GetEnableRecovery() {
		result = append(result, recovery.Recovery())
	}
	if config.GetEnableTracing() {
		result = append(result, tracing.Client())
	}
	if config.GetEnableMetadata() {
		result = append(result, metadata.Client())
	}
	if len(staticMetadata) > 0 {
		result = append(result, clientMetadata(staticMetadata))
	}
	if config.GetEnableLogging() {
		result = append(result, clientLogging())
	}
	if config.GetEnableCircuitBreaker() {
		result = append(result, circuitbreaker.Client())
	}

	authConfig := config.GetAuth()
	if authConfig == nil {
		return append(result, custom...), nil
	}
	secret := authConfig.GetSecret()
	if secret == "" {
		keyValue := sdk.Runtime.GetKey()
		if keyValue == nil {
			return nil, fmt.Errorf("客户端 JWT 密钥为空且运行时密钥未初始化")
		}
		var err error
		var derived []byte
		derived, err = keyValue.Derive(context.Background(), jwtKeyName)
		if err != nil {
			return nil, fmt.Errorf("派生客户端 JWT 密钥失败: %w", err)
		}
		secret = string(derived)
	}
	authenticator, err := jwt.NewAuthenticator(
		jwt.WithKey([]byte(secret)),
		jwt.WithSigningMethod(authConfig.GetMethod()),
	)
	if err != nil {
		return nil, fmt.Errorf("create JWT authenticator: %w", err)
	}
	result = append(result, authnMiddleware.Client(authenticator))
	return append(result, custom...), nil
}

// clientMetadata 将配置中的静态 metadata 写入客户端 transport 请求头。
func clientMetadata(values map[string]string) middleware.Middleware {
	metadataValues := make(map[string]string, len(values))
	for key, value := range values {
		metadataValues[key] = value
	}
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			if clientTransport, ok := transport.FromClientContext(ctx); ok {
				for key, value := range metadataValues {
					clientTransport.RequestHeader().Set(key, value)
				}
			}
			return next(ctx, req)
		}
	}
}

// clientLogging 记录客户端 gRPC 调用的耗时和状态。
func clientLogging() middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			start := time.Now()
			response, err := next(ctx, req)
			level := log.LevelInfo
			code := codes.OK
			if err != nil {
				code = status.Code(err)
				if code >= codes.Internal {
					level = log.LevelError
				} else if code >= codes.NotFound {
					level = log.LevelWarn
				}
			}
			logger := log.Default()
			if logger.Enabled(ctx, level) {
				operation := ""
				endpoint := ""
				if clientTransport, ok := transport.FromClientContext(ctx); ok {
					operation = clientTransport.Operation()
					endpoint = clientTransport.Endpoint()
				}
				args := []any{
					"method", operation,
					"target", endpoint,
					"latency_ms", time.Since(start).Milliseconds(),
					"code", code.String(),
				}
				if err != nil {
					args = append(args, "error", err.Error())
				}
				logger.Log(ctx, level, "grpc client rpc", args...)
			}
			return response, err
		}
	}
}
