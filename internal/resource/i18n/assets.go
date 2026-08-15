package i18n

import (
	"embed"
	"io/fs"
)

// 提供 Core 通用国际化文案的内嵌文件系统。
//
//go:embed assets/*.json
var localesData embed.FS

// Assets 返回 Core 通用国际化文件系统，交由 Core 统一注册和执行。
func Assets() fs.FS {
	value, err := fs.Sub(localesData, "assets")
	if err != nil {
		panic(err)
	}
	return value
}
