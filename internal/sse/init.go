package sse

import "github.com/google/wire"

// ProviderSet 提供 Core SSE 注册、传输和业务运行时。
var ProviderSet = wire.NewSet(NewRegistry, NewSSE, NewStreamResolver, NewTransport)
