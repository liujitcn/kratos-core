package data

import "context"

// APIRecord 表示 OpenAPI 解析后待保存的接口快照。
type APIRecord struct {
	// ID 是接口编号，用于保留已有运行时配置。
	ID int64
	// ToolName 是工具名称。
	ToolName string
	// ToolPrompts 是工具提示词 JSON。
	ToolPrompts string
	// ServiceName 是服务名称。
	ServiceName string
	// ServiceDesc 是服务描述。
	ServiceDesc string
	// Desc 是接口描述。
	Desc string
	// Operation 是 RPC 操作标识。
	Operation string
	// Method 是 HTTP 请求方法。
	Method string
	// Path 是 HTTP 路径。
	Path string
	// McpStatus 是 MCP 工具状态。
	McpStatus int32
	// AgentStatus 是 Agent 工具状态。
	AgentStatus int32
}

// APITranslationRecord 表示一个接口的单语言展示文本。
type APITranslationRecord struct {
	// Operation 是 RPC 操作标识。
	Operation string
	// Locale 是语言区域。
	Locale string
	// ToolPrompts 是工具提示词 JSON。
	ToolPrompts string
	// ServiceDesc 是服务描述。
	ServiceDesc string
	// Desc 是接口描述。
	Desc string
}

// APIPolicyRecord 表示权限重建所需的接口字段。
type APIPolicyRecord struct {
	// Operation 是 RPC 操作标识。
	Operation string
	// Method 是 HTTP 请求方法。
	Method string
}

// APIStore 提供资源同步和权限重建所需的接口数据能力。
type APIStore interface {
	// ReplaceAll 替换当前 OpenAPI 接口快照。
	ReplaceAll(context.Context, []*APIRecord) error
	// ReplaceAllTranslations 替换当前 OpenAPI 接口翻译快照。
	ReplaceAllTranslations(context.Context, []*APITranslationRecord) error
	// ListForPolicy 查询权限重建所需的接口字段。
	ListForPolicy(context.Context) ([]*APIPolicyRecord, error)
}
