package docs

import (
	"github.com/google/wire"
)

// ProviderSet 提供项目文档注册表和查询能力。
var ProviderSet = wire.NewSet(
	NewRegistry,
)
