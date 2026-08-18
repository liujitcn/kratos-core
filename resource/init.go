package resource

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/resource/biz"
	"github.com/liujitcn/kratos-core/resource/docs"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"github.com/liujitcn/kratos-core/resource/migration"
	"github.com/liujitcn/kratos-core/resource/openapi"
)

// ProviderSet 汇总文档、国际化、OpenAPI 和迁移资源，并生成启动期同步结果。
var ProviderSet = wire.NewSet(
	biz.ProviderSet,
	docs.ProviderSet,
	i18n.ProviderSet,
	openapi.ProviderSet,
	migration.ProviderSet,
	NewSyncResult,
)
