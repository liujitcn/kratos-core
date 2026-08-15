package sse

import (
	"context"
	"encoding/json"

	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

// Publisher 将结构化消息发布到已声明的 SSE 流。
type Publisher struct {
	server *sseServer.Server
}

// NewPublisher 创建 SSE JSON 发布器，服务未启用时返回 nil。
func NewPublisher(server *sseServer.Server) *Publisher {
	if server == nil {
		return nil
	}
	return &Publisher{server: server}
}

// TryPublishJSON 编码并尽力发布一条 SSE JSON 消息。
func (p *Publisher) TryPublishJSON(ctx context.Context, streamID, eventID string, payload any) {
	if p == nil || p.server == nil || streamID == "" || eventID == "" {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	p.server.TryPublish(ctx, sseServer.StreamID(streamID), &sseServer.Event{
		Event: []byte(eventID),
		Data:  data,
	})
}
