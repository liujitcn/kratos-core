package data

import "github.com/google/wire"

// ProviderSet 创建 Core 使用的多数据源客户端。
var ProviderSet = wire.NewSet(
	NewClients,
)
