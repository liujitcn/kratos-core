package kratoscore

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
	internalBiz "github.com/liujitcn/kratos-core/internal/biz"
	"github.com/liujitcn/kratos-core/internal/job"
	"github.com/liujitcn/kratos-core/internal/queue"
	"github.com/liujitcn/kratos-core/internal/resource"
	"github.com/liujitcn/kratos-core/internal/resource/migration"
	"github.com/liujitcn/kratos-core/internal/resource/openapi"
	"github.com/liujitcn/kratos-core/internal/server"
	"github.com/liujitcn/kratos-core/internal/sse"
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// Module 是 Core 对外暴露的宿主模块契约别名。
type Module = module.Module

// ProviderSet 是 Core 对外唯一的 Wire 注入入口。
var ProviderSet = wire.NewSet(NewApplication)

// newServers 按应用启动顺序合并 Core 传输服务和后台运行时。
func newServers(
	httpServer kratosTransport.Server,
	grpcServer *kratosGRPC.Server,
	mcpRuntime *server.MCPRuntime,
	sseTransport *sse.Transport,
	queue *queue.Queue,
	jobRuntime *job.Runtime,
) []kratosTransport.Server {
	servers := make([]kratosTransport.Server, 0, 6)
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	if mcpRuntime != nil && mcpRuntime.Server != nil && !mcpRuntime.InProcess {
		servers = append(servers, mcpRuntime.Server)
	}
	if sseTransport != nil && sseTransport.Server != nil && !sseTransport.InProcess {
		servers = append(servers, sseTransport.Server)
	}
	if queue != nil {
		servers = append(servers, queue)
	}
	if jobRuntime != nil {
		servers = append(servers, jobRuntime)
	}
	return servers
}

// newApplication 按“迁移、资源同步、传输绑定、应用创建”的顺序完成启动组装。
func newApplication(
	ctx *bootstrap.Context,
	servers []kratosTransport.Server,
	migrations *migration.Migration,
	openAPIRegistry *openapi.Registry,
	baseAPICase *internalBiz.BaseAPICase,
	baseTenantCase *internalBiz.BaseTenantCase,
	casbinRuleCase *internalBiz.CasbinRuleCase,
	sseRuntime *sse.SSE,
	sseTransport *sse.Transport,
	_ *biz.BaseCase,
) (*kratos.App, func(), error) {
	// 1. 迁移数据库，并按 API、租户菜单、Casbin 规则的依赖顺序同步资源。
	cleanup, err := startInitializers(ctx.Context(), migrations, openAPIRegistry, baseAPICase, baseTenantCase, casbinRuleCase)
	if err != nil {
		return nil, nil, err
	}
	// 2. 资源就绪后绑定进程内 SSE 传输。
	if sseRuntime != nil && sseTransport != nil {
		sseRuntime.BindTransport(sseTransport.Server)
	}
	// 3. 最后交给 Kratos 统一管理服务启动和停止。
	return bootstrap.NewApp(ctx, servers...), newApplicationCleanup(servers, cleanup, sseRuntime, sseTransport), nil
}

// startInitializers 按数据库迁移、OpenAPI、Casbin 的顺序同步执行启动初始化。
func startInitializers(
	ctx context.Context,
	migrations *migration.Migration,
	openAPIRegistry *openapi.Registry,
	baseAPICase *internalBiz.BaseAPICase,
	baseTenantCase *internalBiz.BaseTenantCase,
	casbinRuleCase *internalBiz.CasbinRuleCase,
) (func(), error) {
	var err error
	if migrations != nil {
		err = migrations.Start(ctx)
		if err != nil {
			return nil, err
		}
	}
	if baseAPICase == nil || baseTenantCase == nil || casbinRuleCase == nil {
		if migrations != nil {
			_ = migrations.Stop(context.Background())
		}
		return nil, errors.New("启动资源同步组件未初始化")
	}
	var documents []openapi.Document
	if openAPIRegistry != nil {
		documents = openAPIRegistry.Documents()
	}
	err = resource.SyncAccessControl(ctx, documents, baseAPICase, baseTenantCase, casbinRuleCase)
	if err != nil {
		if migrations != nil {
			_ = migrations.Stop(context.Background())
		}
		return nil, fmt.Errorf("同步启动资源: %w", err)
	}
	return func() {
		if migrations != nil {
			_ = migrations.Stop(context.Background())
		}
	}, nil
}

// newApplicationCleanup 创建按注册逆序停止全部应用服务的清理函数。
func newApplicationCleanup(
	servers []kratosTransport.Server,
	cleanup func(),
	sseRuntime *sse.SSE,
	sseTransport *sse.Transport,
) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			for index := len(servers) - 1; index >= 0; index-- {
				if servers[index] != nil {
					_ = servers[index].Stop(context.Background())
				}
			}
			if cleanup != nil {
				cleanup()
			}
			if sseTransport != nil && sseTransport.InProcess && sseTransport.Server != nil {
				_ = sseTransport.Server.Stop(context.Background())
			}
			if sseRuntime != nil {
				sseRuntime.UnbindTransport()
			}
		})
	}
}
