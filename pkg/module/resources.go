package module

import "io/fs"

// Resources 保存宿主在业务对象构建前向 Core 提供的静态资源。
type Resources struct {
	// ProjectKey 是宿主项目的稳定标识，文档与 OpenAPI 共用该命名空间；留空时使用 kratos-core。
	ProjectKey string
	// ProjectName 是宿主项目展示名称，留空时使用 ProjectKey。
	ProjectName string
	// Models 是宿主数据库自动迁移所需的模型。
	Models Models
	// Docs 是宿主项目文档生成器输出的文件系统，通常包含 docs.json。
	Docs fs.FS
	// I18n 是宿主项目语言 JSON 文件系统，文件名应使用 locale 标识。
	I18n fs.FS
	// OpenAPI 是宿主项目生成的 Swagger/OpenAPI 文件系统集合，通常包含 openapi.yaml。
	OpenAPI fs.FS
	// Migrations 是宿主提供的数据库迁移资源。
	Migrations Migrations
}
