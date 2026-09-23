package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
	"github.com/liujitcn/kratos-kit/transport/queue"
)

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queueName queue.Stream, data any) bool {
	queueID := string(queueName)
	q := sdk.Runtime.GetQueue()
	if q == nil {
		return false
	}
	rawBody, err := json.Marshal(data)
	if err != nil {
		log.Error(fmt.Sprintf("build queue message data error, %s", err.Error()))
		return false
	}
	err = q.Append(queueID, queueData.Message{Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		log.Error(fmt.Sprintf("Append message error, %s", err.Error()))
		return false
	}
	return true
}

// ScheduleQueue 按指定时间向运行时队列安排一条延迟消息。
func ScheduleQueue(ctx context.Context, queueName queue.Stream, messageID string, executeAt time.Time, value any) error {
	queueID := string(queueName)
	q := sdk.Runtime.GetQueue()
	if q == nil {
		return fmt.Errorf("queue is not initialized")
	}
	delayed, ok := q.(interface {
		Schedule(string, queueData.Message, time.Time) error
	})
	if !ok {
		return fmt.Errorf("queue does not support delayed messages")
	}
	rawBody, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("build delayed queue message data: %w", err)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return delayed.Schedule(queueID, queueData.Message{
		ID:     messageID,
		Values: map[string]interface{}{"data": string(rawBody)},
	}, executeAt)
}

// CancelQueue 取消运行时队列中尚未触发的延迟消息。
func CancelQueue(ctx context.Context, queueName queue.Stream, messageID string) error {
	q := sdk.Runtime.GetQueue()
	if q == nil {
		return fmt.Errorf("queue is not initialized")
	}
	delayed, ok := q.(interface {
		Cancel(string, string) error
	})
	if !ok {
		return fmt.Errorf("queue does not support delayed messages")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return delayed.Cancel(string(queueName), messageID)
}
