package sse

import (
	"github.com/google/wire"
)

// ProviderSet 提供 Core SSE 注册、服务和业务运行时。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewStreamResolver,
	NewServer,
)
