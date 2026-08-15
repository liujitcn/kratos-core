package job

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v3/log"
	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	cronTransport "github.com/liujitcn/kratos-kit/transport/cron"
)

var _ kratosTransport.Server = (*Server)(nil)

// Server 将持久化任务调度接入 Kratos 应用生命周期。
type Server struct {
	server    *Scheduler
	transport *cronTransport.Server
	mu        sync.Mutex
	runCancel context.CancelFunc
}

// NewServer 创建任务生命周期服务，模块任务在 Registry 创建阶段注册。
func NewServer(server *Scheduler) *Server {
	if server == nil {
		return nil
	}
	transport := server.Server()
	if transport == nil {
		return nil
	}
	return &Server{server: server, transport: transport}
}

// Start 启动定时任务服务并重载已启用的持久化任务。
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.server == nil || s.transport == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	err = s.transport.Start(ctx)
	if err != nil {
		return err
	}
	runContext, cancel := context.WithCancel(ctx)
	err = s.server.Reload(runContext)
	if err != nil {
		cancel()
		stopErr := s.transport.Stop(ctx)
		s.transport.RemoveAllJobs()
		if stopErr != nil {
			log.Error(fmt.Sprintf("cron server stop failed, err=%v", stopErr))
		}
		s.server.ClearExecutionContext()
		return err
	}
	s.runCancel = cancel
	return nil
}

// Stop 停止定时任务服务并清理已注册任务。
func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.transport == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runCancel != nil {
		s.runCancel()
		s.runCancel = nil
	}
	err := s.transport.Stop(ctx)
	s.transport.RemoveAllJobs()
	if s.server != nil {
		s.server.ClearExecutionContext()
	}
	return err
}
