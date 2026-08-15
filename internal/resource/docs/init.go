package docs

import (
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/pkg/biz"
)

// ProviderSet 提供项目文档注册表和查询能力。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewDocs,
	wire.Bind(new(biz.Docs), new(*Docs)),
)
