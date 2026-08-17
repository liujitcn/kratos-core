package server

import (
	"github.com/google/wire"
)

// ProviderSet 汇总中间件层依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewHTTPMiddleware,
	NewGRPCMiddleware,
	NewGRPCServer,
	NewHTTPServer,
)
