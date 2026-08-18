package module

import "github.com/google/wire"

// ProviderSet 收集宿主模块，并派生模型、文档、国际化、OpenAPI 和迁移资源快照。
var ProviderSet = wire.NewSet(
	NewModules,
	NewResources,
	NewModels,
	NewDocsFromResources,
	NewI18nFromResources,
	NewOpenAPIFromResources,
	NewMigrations,
)
