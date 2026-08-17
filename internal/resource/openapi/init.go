package openapi

import (
	"github.com/google/wire"
)

// ProviderSet 提供 OpenAPI 注册表和查询能力。
var ProviderSet = wire.NewSet(
	NewRegistry,
)
