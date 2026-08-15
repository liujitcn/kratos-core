package module

import "github.com/google/wire"

// ProviderSet 提供模块资源聚合所需的 Wire 构造器。
var ProviderSet = wire.NewSet(
	NewModules,
	NewResources,
	NewModels,
	NewDocsFromResources,
	NewI18nFromResources,
	NewOpenAPIFromResources,
	NewMigrations,
)
