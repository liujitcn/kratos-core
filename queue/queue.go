package queue

import (
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queueName queueTransport.Stream, data any) bool {
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
