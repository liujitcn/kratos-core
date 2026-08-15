package biz

import "context"

// SSE 定义 Core 提供给宿主业务模块的 SSE 订阅与发布能力。
type SSE interface {
	// Serve 校验当前 HTTP 请求并建立指定业务流的 SSE 订阅。
	Serve(context.Context, string, string) error
	// PublishEnabled 判断当前应用是否已启用 SSE 消息发布。
	PublishEnabled() bool
	// PublishJSON 尽力向指定 SSE 流发布结构化消息。
	PublishJSON(context.Context, string, string, any)
}
