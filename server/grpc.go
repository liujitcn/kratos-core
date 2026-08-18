package server

import (
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/resource/i18n"
	coreMiddleware "github.com/liujitcn/kratos-core/server/middleware"
	"github.com/liujitcn/kratos-core/server/middleware/logging"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/cache"
	servergrpc "github.com/liujitcn/kratos-kit/server/grpc"
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
	jwtCfg *configv1.Authentication_Jwt,
	cache cache.Cache,
	catalog *i18n.I18n,
) GRPCMiddlewares {
	var grpcMiddlewares GRPCMiddlewares
	cfg := ctx.GetConfig()
	// i18n国际化
	if i18nMiddleware := coreMiddleware.NewI18nCatalogMiddleware(catalog, cache); i18nMiddleware != nil {
		grpcMiddlewares = append(grpcMiddlewares, i18nMiddleware)
	}
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.EnableLogging && baseUserRepo != nil && authenticator != nil {
		grpcMiddlewares = append(grpcMiddlewares, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	if authenticator != nil && authorizer != nil && userToken != nil && jwtCfg != nil {
		grpcMiddlewares = append(grpcMiddlewares, coreMiddleware.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	}
	// 按 gRPC 服务配置挂载 Core 的校验错误转换，避免未启用时处理校验错误。
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.GetEnableValidate() {
		grpcMiddlewares = append(grpcMiddlewares, coreMiddleware.NewValidateMiddleware())
	}
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

	srv, err := servergrpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	modules.RegisterGRPC(srv)

	return srv, nil
}
