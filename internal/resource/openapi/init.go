package openapi

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/pkg/biz"
)

// ProviderSet 提供 OpenAPI 注册表和查询能力。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewOpenAPI,
	wire.Bind(new(biz.OpenAPI), new(*OpenAPI)),
)
