package i18n

import "github.com/google/wire"

// ProviderSet 合并模块语言资源并创建 Core 国际化消息目录。
var ProviderSet = wire.NewSet(NewCatalog)
