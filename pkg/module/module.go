package module

import (
	"io/fs"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
	mcpserver "github.com/liujitcn/kratos-kit/transport/mcp"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
	sseTransport "github.com/liujitcn/kratos-kit/transport/sse"
	"google.golang.org/grpc"
)

// Module 定义宿主业务模块向 Core 提供的协议注册和静态资源。
type Module interface {
	// RegisterGRPC 将模块 gRPC 服务注册到宿主。
	RegisterGRPC(grpc.ServiceRegistrar)
	// RegisterHTTP 将模块 HTTP 服务注册到宿主。
	RegisterHTTP(*kratosHTTP.Server)
	// RegisterMCP 将模块 MCP 工具注册到宿主。
	RegisterMCP(*mcpserver.Server)
	// RegisterQueue 将模块队列消费者注册到宿主。
	RegisterQueue(*queueTransport.Server)
	// RegisterCron 将模块数据库任务执行器注册到 Cron Server。
	RegisterCron(*cronTransport.Server) error
	// RegisterSSE 将模块业务 SSE 流注册到 SSE Server。
	RegisterSSE(*sseTransport.Server) error
	// Resources 返回模块提供的文档、迁移、国际化和 OpenAPI 资源。
	Resources() Resources
}

// Modules 聚合多个宿主业务模块，并统一转发协议注册。
type Modules []Module

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

// NewModules 将 Wire 传入的模块参数收集为 Core 运行时模块集合。
func NewModules(modules ...Module) Modules {
	result := make(Modules, 0, len(modules))
	for _, item := range modules {
		if item != nil {
			result = append(result, item)
		}
	}
	return result
}

// NewResources 收集全部模块的一次性资源快照。
func NewResources(modules Modules) []Resources {
	result := make([]Resources, 0, len(modules))
	for _, item := range modules {
		if item != nil {
			result = append(result, item.Resources())
		}
	}
	return result
}

// NewModels 按数据源名称汇总模块资源快照中的数据库模型。
func NewModels(resources []Resources) Models {
	models := Models{}
	for _, resource := range resources {
		for name, values := range resource.Models {
			models[name] = append(models[name], values...)
		}
	}
	return models
}

// NewModelsFromModules 收集模块资源并按数据源名称汇总数据库模型。
func NewModelsFromModules(modules Modules) Models {
	return NewModels(NewResources(modules))
}

// NewDocs 收集全部模块的项目文档文件系统。
func NewDocs(modules Modules) Docs {
	return NewDocsFromResources(NewResources(modules))
}

// NewDocsFromResources 收集模块资源快照中的项目文档文件系统。
func NewDocsFromResources(resources []Resources) Docs {
	docs := Docs{}
	for _, resource := range resources {
		if resource.Docs != nil {
			docs = append(docs, ResourcesItem{
				FS:          resource.Docs,
				ProjectKey:  resource.ProjectKey,
				ProjectName: resource.ProjectName,
			})
		}
	}
	return docs
}

// NewOpenAPI 收集全部模块的 OpenAPI 文件系统。
func NewOpenAPI(modules Modules) OpenAPI {
	return NewOpenAPIFromResources(NewResources(modules))
}

// NewOpenAPIFromResources 收集模块资源快照中的 OpenAPI 文件系统。
func NewOpenAPIFromResources(resources []Resources) OpenAPI {
	openAPI := OpenAPI{}
	for _, resource := range resources {
		if resource.OpenAPI != nil {
			openAPI = append(openAPI, ResourcesItem{
				FS:          resource.OpenAPI,
				ProjectKey:  resource.ProjectKey,
				ProjectName: resource.ProjectName,
			})
		}
	}
	return openAPI
}

// NewI18n 收集全部模块的国际化文件系统。
func NewI18n(modules Modules) I18n {
	return NewI18nFromResources(NewResources(modules))
}

// NewI18nFromResources 收集模块资源快照中的国际化文件系统。
func NewI18nFromResources(resources []Resources) I18n {
	i18n := I18n{}
	for _, resource := range resources {
		if resource.I18n != nil {
			i18n = append(i18n, ResourcesItem{
				FS:          resource.I18n,
				ProjectKey:  resource.ProjectKey,
				ProjectName: resource.ProjectName,
			})
		}
	}
	return i18n
}

// NewMigrations 收集模块资源快照中的数据库迁移资源。
func NewMigrations(resources []Resources) Migrations {
	migrations := make(Migrations, 0, len(resources))
	for _, resource := range resources {
		migrations = append(migrations, resource.Migrations...)
	}
	return migrations
}

// NewMigrationsFromModules 收集模块资源中的数据库迁移资源。
func NewMigrationsFromModules(modules Modules) Migrations {
	return NewMigrations(NewResources(modules))
}

// RegisterGRPC 将全部模块 gRPC 服务注册到宿主。
func (modules Modules) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	for _, module := range modules {
		if module != nil {
			module.RegisterGRPC(registrar)
		}
	}
}

// RegisterHTTP 将全部模块 HTTP 路由注册到宿主。
func (modules Modules) RegisterHTTP(server *kratosHTTP.Server) {
	for _, module := range modules {
		if module != nil {
			module.RegisterHTTP(server)
		}
	}
}

// RegisterMCP 将全部模块 MCP 工具注册到宿主。
func (modules Modules) RegisterMCP(server *mcpserver.Server) {
	for _, module := range modules {
		if module != nil {
			module.RegisterMCP(server)
		}
	}
}

// RegisterQueue 将全部模块队列消费者注册到宿主。
func (modules Modules) RegisterQueue(server *queueTransport.Server) {
	for _, module := range modules {
		if module != nil {
			module.RegisterQueue(server)
		}
	}
}

// RegisterCron 将全部模块数据库任务执行器注册到 Cron Server。
func (modules Modules) RegisterCron(server *cronTransport.Server) error {
	for _, module := range modules {
		if module != nil {
			if err := module.RegisterCron(server); err != nil {
				return err
			}
		}
	}
	return nil
}

// RegisterSSE 将全部模块业务 SSE 流注册到 SSE Server。
func (modules Modules) RegisterSSE(server *sseTransport.Server) error {
	for _, module := range modules {
		if module != nil {
			if err := module.RegisterSSE(server); err != nil {
				return err
			}
		}
	}
	return nil
}
