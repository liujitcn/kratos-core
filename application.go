package kratoscore

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/log"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/internal/job"
	"github.com/liujitcn/kratos-core/internal/mcp"
	"github.com/liujitcn/kratos-core/internal/queue"
	"github.com/liujitcn/kratos-core/internal/resource"
	"github.com/liujitcn/kratos-core/internal/sse"
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// Module 是 Core 对外暴露的宿主模块契约别名。
type Module = module.Module

// ProviderSet 是 Core 对外唯一的 Wire 注入入口。
var ProviderSet = wire.NewSet(NewApplicationWithModule)

// NewApplicationWithModule 通过 Core ProviderSet 装配单个业务模块。
func NewApplicationWithModule(ctx *bootstrap.Context, mod Module) (*kratos.App, func(), error) {
	return NewApplication(ctx, mod)
}

// newServers 按应用启动顺序合并 Core 传输服务和后台运行时。
func newServers(
	httpServer kratosTransport.Server,
	grpcServer *kratosGRPC.Server,
	mcpServer *mcp.Server,
	sseServer *sse.Server,
	queueServer *queue.Server,
	jobServer *job.Server,
) []kratosTransport.Server {
	servers := make([]kratosTransport.Server, 0, 6)
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	if mcpServer != nil && mcpServer.Server != nil && !mcpServer.InProcess {
		servers = append(servers, mcpServer)
	}
	if sseServer != nil && sseServer.Server != nil && !sseServer.InProcess {
		servers = append(servers, sseServer)
	}
	if queueServer != nil {
		servers = append(servers, queueServer)
	}
	if jobServer != nil {
		servers = append(servers, jobServer)
	}
	return servers
}

// newApplication 在资源构造完成后创建 Kratos 应用。
func newApplication(
	ctx *bootstrap.Context,
	_ *resource.PermissionSynchronizer,
	servers []kratosTransport.Server,
	sseServer *sse.Server,
	_ *biz.BaseCase,
	// 这些接口的实现位于 internal，Core 启动图必须保留组装，宿主不能直接构造实现。
	_ biz.Job,
	_ biz.Docs,
	_ biz.OpenAPI,
	_ biz.SSE,
) (*kratos.App, func(), error) {
	// 迁移、API、租户菜单和 Casbin 规则已在对应资源构造阶段完成。
	// 由 Kratos 统一管理服务启动和停止。
	app := bootstrap.NewApp(ctx, servers...)
	var once sync.Once
	return app, func() {
		once.Do(func() {
			for index := len(servers) - 1; index >= 0; index-- {
				if servers[index] != nil {
					stopErr := servers[index].Stop(context.Background())
					if stopErr != nil {
						log.Error(fmt.Sprintf("停止 Core 服务失败: %v", stopErr))
					}
				}
			}
			if sseServer != nil && sseServer.InProcess && sseServer.Server != nil {
				stopErr := sseServer.Stop(context.Background())
				if stopErr != nil {
					log.Error(fmt.Sprintf("停止进程内 SSE 服务失败: %v", stopErr))
				}
			}
		})
	}, nil
}
