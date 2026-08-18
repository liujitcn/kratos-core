package sse

import (
	"github.com/google/wire"
)

// ProviderSet 创建 SSE 注册表、流解析器、传输服务和业务运行时。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewStreamResolver,
	NewServer,
	NewSSE,
)
