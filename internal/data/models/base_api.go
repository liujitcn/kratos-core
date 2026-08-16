package models

import "gorm.io/plugin/soft_delete"

// TableNameBaseAPI 是 API 表名。
const TableNameBaseAPI = "base_api"

// BaseAPI 保存 Core 根据 OpenAPI 文档同步的接口信息。
type BaseAPI struct {
	ID          int64                 `gorm:"column:id;type:bigint;primaryKey;autoIncrement:true;comment:API ID" json:"id"`
	ToolName    string                `gorm:"column:tool_name;type:varchar(150);not null;index:idx_base_api_tool_name,priority:1;comment:工具名" json:"tool_name"`
	ToolPrompts string                `gorm:"column:tool_prompts;type:json;not null;comment:工具提示词" json:"tool_prompts"`
	ServiceName string                `gorm:"column:service_name;type:varchar(50);not null;comment:服务名" json:"service_name"`
	ServiceDesc string                `gorm:"column:service_desc;type:varchar(50);not null;comment:服务描述" json:"service_desc"`
	Desc        string                `gorm:"column:desc;type:varchar(500);not null;comment:描述" json:"desc"`
	Operation   string                `gorm:"column:operation;type:varchar(255);not null;uniqueIndex:uk_base_api_operation;comment:操作方法" json:"operation"`
	Method      string                `gorm:"column:method;type:varchar(10);not null;comment:请求方式" json:"method"`
	Path        string                `gorm:"column:path;type:varchar(512);not null;comment:请求地址" json:"path"`
	McpStatus   int32                 `gorm:"column:mcp_status;type:tinyint;not null;index:idx_base_api_mcp_status,priority:1;comment:MCP工具状态：枚举【Status】" json:"mcp_status"`
	AgentStatus int32                 `gorm:"column:agent_status;type:tinyint;not null;index:idx_base_api_agent_status,priority:1;comment:Agent工具状态：枚举【Status】" json:"agent_status"`
	DeletedAt   soft_delete.DeletedAt `gorm:"column:deleted_at;type:bigint unsigned;not null;comment:删除时间;softDelete:milli" json:"deleted_at"`
}

// TableName 返回 API 表名。
func (*BaseAPI) TableName() string { return TableNameBaseAPI }

// TableComment 返回 API 表注释。
func (*BaseAPI) TableComment() string { return "API信息" }
