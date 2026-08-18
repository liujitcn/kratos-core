package mcp

import "github.com/google/wire"

// ProviderSet 创建 MCP 运行服务，并注册宿主模块提供的 MCP 工具。
var ProviderSet = wire.NewSet(NewServer)
