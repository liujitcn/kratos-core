package mcp

import "github.com/google/wire"

// ProviderSet 提供 Core MCP 服务。
var ProviderSet = wire.NewSet(NewServer)
