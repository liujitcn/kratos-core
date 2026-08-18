package server

import (
	"github.com/google/wire"
)

// ProviderSet 创建 HTTP 与 gRPC 的中间件链及可选协议服务器。
var ProviderSet = wire.NewSet(
	NewHTTPMiddleware,
	NewGRPCMiddleware,
	NewGRPCServer,
	NewHTTPServer,
)
