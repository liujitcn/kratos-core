package queue

import (
	"bytes"
	"encoding/json"
	"fmt"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// Decode 从队列消息的 data 字段解码业务对象。
func Decode[T any](message queueData.Message) (*T, error) {
	raw, exists := message.Values["data"]
	if !exists || raw == nil {
		return nil, fmt.Errorf("队列消息缺少 data 字段")
	}
	var data []byte
	var err error
	switch value := raw.(type) {
	case string:
		data = []byte(value)
	case []byte:
		data = value
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, fmt.Errorf("队列消息 data 为空")
	}
	var value T
	if err = json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}
