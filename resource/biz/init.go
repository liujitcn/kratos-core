package biz

import "github.com/google/wire"

// ProviderSet 创建 API、租户和 Casbin 规则等启动期资源同步业务。
var ProviderSet = wire.NewSet(
	NewBaseAPICase,
	NewBaseTenantCase,
	NewCasbinRuleCase,
)
