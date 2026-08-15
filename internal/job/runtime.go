package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
)

var _ kratosTransport.Server = (*Runtime)(nil)

// Runtime 将持久化任务调度接入 Kratos 应用生命周期。
type Runtime struct {
	server    *Scheduler
	transport *cronTransport.Server
	mu        sync.Mutex
	runCancel context.CancelFunc
}

// NewRuntime 创建任务生命周期运行时，模块任务在 Registry 创建阶段注册。
func NewRuntime(server *Scheduler) *Runtime {
	if server == nil {
		return nil
	}
	transport := server.TransportServer()
	if transport == nil {
		return nil
	}
	return &Runtime{server: server, transport: transport}
}

// Start 启动定时任务服务并重载已启用的持久化任务。
func (r *Runtime) Start(ctx context.Context) error {
	if r == nil || r.server == nil || r.transport == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var err error
	err = r.transport.Start(ctx)
	if err != nil {
		return err
	}
	runContext, cancel := context.WithCancel(ctx)
	err = r.server.Reload(runContext)
	if err != nil {
		cancel()
		stopErr := r.transport.Stop(ctx)
		r.transport.RemoveAllJobs()
		if stopErr != nil {
			log.Error(fmt.Sprintf("cron server stop failed, err=%v", stopErr))
		}
		r.server.ClearExecutionContext()
		return err
	}
	r.runCancel = cancel
	return nil
}

// Stop 停止定时任务服务并清理已注册任务。
func (r *Runtime) Stop(ctx context.Context) error {
	if r == nil || r.transport == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runCancel != nil {
		r.runCancel()
		r.runCancel = nil
	}
	err := r.transport.Stop(ctx)
	r.transport.RemoveAllJobs()
	if r.server != nil {
		r.server.ClearExecutionContext()
	}
	return err
}
