package queue

import (
	"context"
	"fmt"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	"github.com/liujitcn/kratos-core/module"
	queue2 "github.com/liujitcn/kratos-core/queue"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

var _ kratosTransport.Server = (*Server)(nil)

// Server 将 Core 队列服务接入 Kratos 应用生命周期。
type Server struct {
	server *queueTransport.Server
}

// NewServer 创建队列服务，注册 Core 和模块提供的队列消费者。
func NewServer(queue kitQueue.Queue, baseJobLogRepository *data.BaseJobLogRepository, baseLogRepository *data.BaseLogRepository, modules module.Modules) (*Server, error) {
	if queue == nil {
		return nil, fmt.Errorf("队列适配器不能为空")
	}
	server, err := queueTransport.NewServer(queueTransport.WithQueue(queue))
	if err != nil {
		return nil, err
	}
	server.Register(_const.JOB_LOG, func(message queueData.Message) error {
		var entity *models.BaseJobLog
		entity, err = queue2.Decode[models.BaseJobLog](message)
		if err != nil {
			return err
		}
		return baseJobLogRepository.Create(context.Background(), entity)
	})
	server.Register(_const.LOG, func(message queueData.Message) error {
		var entity *models.BaseLog
		entity, err = queue2.Decode[models.BaseLog](message)
		if err != nil {
			return err
		}
		return baseLogRepository.Create(context.Background(), entity)
	})

	modules.RegisterQueue(server)
	return &Server{server: server}, nil
}

// Start 启动队列消费服务。
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Start(ctx)
}

// Stop 停止队列消费服务。
func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Stop(ctx)
}
