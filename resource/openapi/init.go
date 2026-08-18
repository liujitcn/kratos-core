package openapi

import (
	"github.com/google/wire"
)

// ProviderSet 合并模块 OpenAPI 文档，并创建注册表和查询入口。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewOpenAPI,
)
