package migration

import "github.com/google/wire"

// ProviderSet 注册并执行宿主模块提供的数据库迁移资源。
var ProviderSet = wire.NewSet(NewMigration)
