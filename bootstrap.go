// Package kratoscore 提供 Core 应用的依赖注入入口与运行时组装能力。
package kratoscore

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/config"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/job"
	"github.com/liujitcn/kratos-core/mcp"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-core/resource"
	"github.com/liujitcn/kratos-core/server"
	"github.com/liujitcn/kratos-core/sse"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// ProviderSet 汇总 Core 应用所需的配置、业务、资源与传输层依赖。
// 宿主只需提供 bootstrap.Context 和业务模块，Wire 会按此集合完成其余运行时组件的装配。
// Wire 根据类型依赖图解析提供者，以下排列仅用于职责分组，不表示实例创建或服务启动顺序。
var ProviderSet = wire.NewSet(
	// 注册配置解析、基础设施客户端、模块资源聚合与数据访问能力。
	config.ProviderSet,
	biz.ProviderSet,
	module.ProviderSet,
	data.ProviderSet,
	// 注册文档、国际化、OpenAPI、迁移等资源能力，以及 SSE、队列、任务和 MCP 运行组件。
	resource.ProviderSet,
	sse.ProviderSet,
	queue.ProviderSet,
	job.ProviderSet,
	mcp.ProviderSet,
	// 注册 HTTP 与 gRPC 的中间件链和协议服务器。
	server.ProviderSet,
	// 将已解析的可选服务汇总为最终 Kratos 应用。
	NewApp,
)

// NewApp 将 Wire 创建的 Core 组件组装为最终的 Kratos 应用。
// syncResult 记录依赖注入阶段已经完成的资源同步结果；HTTP 与 gRPC 服务按配置可选创建；
// MCP 和 SSE 仅在独立传输模式下加入应用生命周期，进程内模式由 HTTP 服务统一承载；
// 队列消费者和定时任务作为后台服务随应用共同启动和停止。
func NewApp(
	ctx *bootstrap.Context,
	syncResult *resource.SyncResult,
	httpServer transport.Server,
	grpcServer *grpc.Server,
	mcpServer *mcp.Server,
	sseServer *sse.Server,
	queueServer *queue.Server,
	jobServer *job.Server,
) *kratos.App {
	// 资源同步已在应用构造前完成，此处只记录最终统计，避免启动后再次执行同步。
	if syncResult != nil {
		log.Info("资源同步完成",
			"documents", syncResult.DocumentCount,
			"apis", syncResult.APICount,
		)
	}

	// 最多挂载 HTTP、gRPC、MCP、SSE、队列和定时任务六类服务；未配置的服务保持 nil 并直接跳过。
	servers := make([]transport.Server, 0, 6)
	// HTTP 和 gRPC 是常规对外协议服务，创建成功后直接交由 Kratos 管理生命周期。
	if httpServer != nil {
		servers = append(servers, httpServer)
	}
	if grpcServer != nil {
		servers = append(servers, grpcServer)
	}
	// 独立 MCP 服务拥有自己的监听端点；进程内 MCP 已挂载到 HTTP Server，不能重复启动。
	if mcpServer != nil && mcpServer.Server != nil && !mcpServer.InProcess {
		servers = append(servers, mcpServer)
	}
	// 独立 SSE 服务由 Kratos 启停；进程内 SSE 与 HTTP Server 共用监听端口和生命周期。
	if sseServer != nil && sseServer.Server != nil && !sseServer.InProcess {
		servers = append(servers, sseServer)
	}
	// 队列消费者和持久化任务调度器都实现 transport.Server，统一参与应用启停和错误传播。
	if queueServer != nil {
		servers = append(servers, queueServer)
	}
	if jobServer != nil {
		servers = append(servers, jobServer)
	}
	// bootstrap 负责注入日志、注册中心等基础能力，并按上述顺序管理所有独立服务。
	return bootstrap.NewApp(ctx, servers...)
}
