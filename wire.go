//go:build wireinject
// +build wireinject

package kratoscore

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	internalBiz "github.com/liujitcn/kratos-core/internal/biz"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/job"
	"github.com/liujitcn/kratos-core/internal/mcp"
	"github.com/liujitcn/kratos-core/internal/queue"
	"github.com/liujitcn/kratos-core/internal/resource"
	"github.com/liujitcn/kratos-core/internal/server"
	"github.com/liujitcn/kratos-core/internal/sse"
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/config"
	"github.com/liujitcn/kratos-core/pkg/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// NewApplication 通过 Core ProviderSet 装配多个业务模块。
func NewApplication(ctx *bootstrap.Context, modules ...module.Module) (*kratos.App, func(), error) {
	panic(wire.Build(
		// 1. 解析配置并收集宿主模块资源，供数据库、资源注册表和服务使用。
		config.ProviderSet,
		module.ProviderSet,
		biz.ProviderSet,
		data.ProviderSet,
		// 2. 创建 Core 资源注册表和运行时传输组件。
		resource.ProviderSet,
		sse.ProviderSet,
		queue.ProviderSet,
		job.ProviderSet,
		mcp.ProviderSet,
		// 3. 创建鉴权、中间件和协议服务器。
		server.ProviderSet,
		// 4. 创建启动期资源同步业务，并组装服务列表和应用生命周期。
		internalBiz.ProviderSet,
		newServers,
		newApplication,
	))
}
