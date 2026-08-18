package docs

import (
	"github.com/google/wire"
)

// ProviderSet 合并模块文档资源，并创建项目文档注册表和查询入口。
var ProviderSet = wire.NewSet(
	NewRegistry,
	NewDocs,
)
