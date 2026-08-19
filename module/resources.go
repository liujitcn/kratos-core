package module

import "io/fs"

// Resource 定义宿主向 Core 提供的一组静态资源。
type Resource interface {
	// ProjectKey 是宿主项目的稳定标识，文档与 OpenAPI 共用该命名空间；留空时使用 kratos-core。
	ProjectKey() string
	// ProjectName 是宿主项目展示名称，留空时使用 ProjectKey。
	ProjectName() string
	// Models 是宿主数据库自动迁移所需的模型。
	Models() Models
	// Docs 是宿主项目文档生成器输出的文件系统，包含 docs.json 和可选的 docs.<locale>.json。
	Docs() fs.FS
	// I18n 是宿主项目语言 JSON 文件系统，文件名应使用 locale 标识。
	I18n() fs.FS
	// OpenAPI 是宿主项目生成的 Swagger/OpenAPI 文件系统集合，包含默认 openapi.yaml 和可选的 openapi.<locale>.yaml。
	OpenAPI() fs.FS
	// Migrations 是宿主提供的数据库迁移资源。
	Migrations() Migrations
}

// Resources 聚合多个宿主提供的静态资源。
type Resources []Resource

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

// Models 按数据源名称保存模块提供的数据库模型。
type Models map[string][]interface{}

// ResourcesItem 描述一组带项目归属信息的文件系统资源。
type ResourcesItem struct {
	// FS 是资源所在的文件系统。
	FS fs.FS
	// ProjectKey 是宿主项目的稳定标识，文档与 OpenAPI 共用该命名空间；留空时使用 kratos-core。
	ProjectKey string
	// ProjectName 是宿主项目展示名称，留空时使用 ProjectKey。
	ProjectName string
}

// Docs 是模块项目文档资源集合。
type Docs []ResourcesItem

// OpenAPI 是模块 OpenAPI 文档资源集合。
type OpenAPI []ResourcesItem

// I18n 是模块国际化文件资源集合。
type I18n []ResourcesItem

// Migrations 是模块数据库迁移资源集合。
type Migrations []Migration

// NewModels 按数据源名称汇总模块资源中的数据库模型。
func NewModels(resources Resources) Models {
	models := Models{}
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		for name, values := range resource.Models() {
			models[name] = append(models[name], values...)
		}
	}
	return models
}

// NewDocsFromResources 收集模块资源中的项目文档文件系统。
func NewDocsFromResources(resources Resources) Docs {
	docs := Docs{}
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		files := resource.Docs()
		if files == nil {
			continue
		}
		docs = append(docs, ResourcesItem{
			FS:          files,
			ProjectKey:  resource.ProjectKey(),
			ProjectName: resource.ProjectName(),
		})
	}
	return docs
}

// NewOpenAPIFromResources 收集模块资源中的 OpenAPI 文件系统。
func NewOpenAPIFromResources(resources Resources) OpenAPI {
	openAPI := OpenAPI{}
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		files := resource.OpenAPI()
		if files == nil {
			continue
		}
		openAPI = append(openAPI, ResourcesItem{
			FS:          files,
			ProjectKey:  resource.ProjectKey(),
			ProjectName: resource.ProjectName(),
		})
	}
	return openAPI
}

// NewI18nFromResources 收集模块资源中的国际化文件系统。
func NewI18nFromResources(resources Resources) I18n {
	i18n := I18n{}
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		files := resource.I18n()
		if files == nil {
			continue
		}
		i18n = append(i18n, ResourcesItem{
			FS:          files,
			ProjectKey:  resource.ProjectKey(),
			ProjectName: resource.ProjectName(),
		})
	}
	return i18n
}

// NewMigrations 收集模块资源中的数据库迁移资源。
func NewMigrations(resources Resources) Migrations {
	migrations := make(Migrations, 0, len(resources))
	for _, resource := range resources {
		if resource != nil {
			migrations = append(migrations, resource.Migrations()...)
		}
	}
	return migrations
}
