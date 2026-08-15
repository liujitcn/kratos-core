package queue

import (
	"context"
	"fmt"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-core/internal/data"
	"github.com/liujitcn/kratos-core/internal/data/models"
	_const "github.com/liujitcn/kratos-core/pkg/const"
	"github.com/liujitcn/kratos-core/pkg/module"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

var _ kratosTransport.Server = (*Queue)(nil)

// Queue 将 Core 队列服务接入 Kratos 应用生命周期。
type Queue struct {
	server *queueTransport.Server
}

// NewQueue 创建队列服务，注册 Core 和模块提供的队列消费者。
func NewQueue(queue kitQueue.Queue, baseJobLogRepository *data.BaseJobLogRepository, baseLogRepository *data.BaseLogRepository, modules module.Modules) (*Queue, error) {
	if queue == nil {
		return nil, fmt.Errorf("队列适配器不能为空")
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
	server.Register(_const.LOG, func(message queueData.Message) error {
		var entity *models.BaseLog
		entity, err = Decode[models.BaseLog](message)
		if err != nil {
			return err
		}
		return baseLogRepository.Create(context.Background(), entity)
	})

	modules.RegisterQueue(server)
	return &Queue{server: server}, nil
}

// Start 启动队列消费服务。
func (q *Queue) Start(ctx context.Context) error {
	if q == nil || q.server == nil {
		return nil
	}
	return q.server.Start(ctx)
}

// Stop 停止队列消费服务。
func (q *Queue) Stop(ctx context.Context) error {
	if q == nil || q.server == nil {
		return nil
	}
	return q.server.Stop(ctx)
}
