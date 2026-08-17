package biz

import "github.com/google/wire"

// ProviderSet 提供 Core 资源同步目录。
var ProviderSet = wire.NewSet(
	NewBaseAPICase,
	NewBaseTenantCase,
	NewCasbinRuleCase,
)
