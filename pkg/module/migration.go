package module

import (
	"io/fs"
)

// Migration 描述一个模块提供的版本化数据库迁移资源。
type Migration struct {
	// Name 是迁移模块的稳定标识。
	Name string
	// FS 是迁移脚本所在的文件系统。
	FS fs.FS
	// Path 是迁移版本目录在文件系统中的根路径。
	Path string
	// Dependencies 是当前迁移模块依赖的其他迁移模块名称。
	Dependencies []string
}
