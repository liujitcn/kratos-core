package server

import (
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/resource/i18n"
	_middleware "github.com/liujitcn/kratos-core/internal/server/middleware"
	"github.com/liujitcn/kratos-core/internal/server/middleware/logging"
	"github.com/liujitcn/kratos-core/pkg/module"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/rpc"
	"github.com/liujitcn/kratos-kit/rpc/middleware/requestid"
)

// GRPCMiddlewares 表示 GRPC 服务中间件链。
type GRPCMiddlewares []middleware.Middleware

// NewGRPCMiddleware 创建 GRPC 服务统一中间件链。
func NewGRPCMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepository,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
	cache cache.Cache,
	catalog *i18n.I18n,
) GRPCMiddlewares {
	var grpcMiddlewares GRPCMiddlewares
	cfg := ctx.GetConfig()
	// 先补齐请求标识，再进入访问日志中间件，确保日志能读取到统一 request_id。
	grpcMiddlewares = append(grpcMiddlewares, requestid.NewRequestIDMiddleware())
	// i18n国际化
	if i18nMiddleware := _middleware.NewI18nCatalogMiddleware(catalog, cache); i18nMiddleware != nil {
		grpcMiddlewares = append(grpcMiddlewares, i18nMiddleware)
	}
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.EnableLogging && baseUserRepo != nil && authenticator != nil {
		grpcMiddlewares = append(grpcMiddlewares, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	if authenticator != nil && authorizer != nil && userToken != nil && jwtCfg != nil {
		grpcMiddlewares = append(grpcMiddlewares, _middleware.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	}
	grpcMiddlewares = append(grpcMiddlewares, _middleware.NewValidateMiddleware())
	return grpcMiddlewares
}

// NewGRPCServer 创建 GRPC Server 并注册已启用业务模块。
func NewGRPCServer(
	ctx *bootstrap.Context,
	middlewares GRPCMiddlewares,
	modules module.Modules,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 GRPC 配置时，跳过 GRPC 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	modules.RegisterGRPC(srv)

	return srv, nil
}
