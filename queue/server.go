package queue

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/audit"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/internal/models"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

var _ transport.Server = (*Server)(nil)

// Server 将 Core 队列服务接入 Kratos 应用生命周期。
type Server struct {
	server *queueTransport.Server
}

// Consumer 描述宿主提供的一项队列消费者。
type Consumer struct {
	// Stream 是消费者监听的队列流名称。
	Stream queueTransport.Stream
	// Handler 是收到队列消息后的处理函数。
	Handler queueData.ConsumerFunc
}

// Consumers 聚合宿主提供的队列消费者。
type Consumers []Consumer

// NewServer 创建队列服务，注册 Core 和宿主提供的队列消费者。
func NewServer(queue kitQueue.Queue, baseJobLogRepository *data.BaseJobLogRepository, auditPipeline *audit.Pipeline, consumers Consumers) (*Server, error) {
	if queue == nil {
		return nil, fmt.Errorf("队列适配器不能为空")
	}
	for _, consumer := range consumers {
		if consumer.Stream == "" {
			return nil, fmt.Errorf("队列流不能为空")
		}
		if consumer.Handler == nil {
			return nil, fmt.Errorf("队列消费者不能为空: %s", consumer.Stream)
		}
	}
	server, err := queueTransport.NewServer(queueTransport.WithQueue(queue))
	if err != nil {
		return nil, err
	}
	server.Register(_const.JOB_LOG, func(message queueData.Message) error {
		var entity *models.BaseJobLog
		entity, err = Decode[models.BaseJobLog](message)
		if err != nil {
			return err
		}
		return baseJobLogRepository.Create(context.Background(), entity)
	})
	server.Register(audit.EventStream, auditPipeline.Consume)
	for _, consumer := range consumers {
		server.Register(consumer.Stream, consumer.Handler)
	}
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
