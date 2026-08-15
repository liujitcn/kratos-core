package sse

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/pkg/biz"
)

// ProviderSet 提供 Core SSE 注册、服务和业务运行时。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewSSE,
	NewStreamResolver,
	NewServer,
	wire.Bind(new(biz.SSE), new(*SSE)),
)
