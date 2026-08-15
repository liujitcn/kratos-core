package i18n

import "github.com/google/wire"

// ProviderSet 提供 Core 国际化目录。
var ProviderSet = wire.NewSet(NewCatalog)
