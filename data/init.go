package data

import "github.com/google/wire"

// ProviderSet 创建多数据源客户端、事务入口及 Core 内置数据仓储。
var ProviderSet = wire.NewSet(
	NewClients,
	NewData,
	NewBaseAPIRepository,
	NewBaseAPII18NRepository,
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
