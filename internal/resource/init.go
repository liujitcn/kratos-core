package resource

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/internal/resource/docs"
	"github.com/liujitcn/kratos-core/internal/resource/i18n"
	"github.com/liujitcn/kratos-core/internal/resource/migration"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
)

// ProviderSet 提供 Core 四类模块资源的统一组装器。
var ProviderSet = wire.NewSet(
	docs.ProviderSet,
	i18n.ProviderSet,
	openapi.ProviderSet,
	migration.NewMigration,
	NewPermissionSynchronizer,
)
