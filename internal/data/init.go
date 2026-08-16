package data

import "github.com/google/wire"

// ProviderSet 提供 Core 业务能力共用的数据访问对象。
var ProviderSet = wire.NewSet(
	NewData,
	NewBaseAPIRepository,
	NewBaseAPII18nRepository,
	NewBaseJobRepository,
	NewBaseJobLogRepository,
	NewBaseLogRepository,
	NewBaseMenuRepository,
	NewBaseRoleRepository,
	NewBaseTenantRepository,
	NewBaseUserRepository,
	NewCasbinRuleRepository,
	wire.Bind(new(Transaction), new(*Data)),
)
