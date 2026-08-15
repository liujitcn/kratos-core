package server

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/internal/server/middleware"
)

// ProviderSet 汇总中间件层依赖注入提供者。
var ProviderSet = wire.NewSet(
	middleware.NewAuthenticator,
	middleware.NewAuthzEngine,
	middleware.NewUserToken,
	NewHTTPMiddleware,
	NewGRPCMiddleware,
	NewGRPCServer,
	NewHTTPServer,
	NewMCPServer,
)
