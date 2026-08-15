package kratoscore

import (
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
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// ProviderSet 是 Core 对外唯一的 Wire 注入入口。
var ProviderSet = wire.NewSet(NewApp)

// newApp 组装应用实例并挂载定时任务、GRPC 与 HTTP 服务。
func newApp(
	ctx *bootstrap.Context,
	syncResult *resource.SyncResult,
	httpServer kratosTransport.Server,
	grpcServer *kratosGRPC.Server,
	mcpServer *mcp.Server,
	sseServer *sse.Server,
	queueServer *queue.Server,
	jobServer *job.Server,
	_ biz.Job,
	_ biz.Docs,
	_ biz.OpenAPI,
	_ biz.SSE,
) *kratos.App {
	// 同步资源结果
	if syncResult != nil {
		log.Info("资源同步完成",
			"documents", syncResult.DocumentCount,
			"apis", syncResult.APICount,
		)
	}
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
	return bootstrap.NewApp(ctx, servers...)
}
