package migration

import "github.com/google/wire"

// ProviderSet 提供 Core 迁移目录。
var ProviderSet = wire.NewSet(NewMigration)
