package models

import "time"

// TableNameBaseAPILog 是 API 访问日志表名。
const TableNameBaseAPILog = "base_api_log"

// BaseAPILog 保存 Core 写入的 API 访问日志字段。
// 完整表结构和自动迁移模型由宿主模块提供。
type BaseAPILog struct {
	ID           int64     `gorm:"column:id" json:"id"`                       // API 日志编号。
	TenantID     int64     `gorm:"column:tenant_id" json:"tenant_id"`         // 租户编号。
	TenantCode   string    `gorm:"column:tenant_code" json:"tenant_code"`     // 租户编码。
	UserID       int64     `gorm:"column:user_id" json:"user_id"`             // 用户编号。
	UserName     string    `gorm:"column:user_name" json:"user_name"`         // 用户名称。
	RequestID    string    `gorm:"column:request_id" json:"request_id"`       // 请求编号。
	TraceID      string    `gorm:"column:trace_id" json:"trace_id"`           // 链路追踪编号。
	ServiceName  string    `gorm:"column:service_name" json:"service_name"`   // 服务名称。
	Operation    string    `gorm:"column:operation" json:"operation"`         // 服务操作标识。
	Method       string    `gorm:"column:method" json:"method"`               // 请求方法。
	Path         string    `gorm:"column:path" json:"path"`                   // 请求路径。
	StatusCode   int32     `gorm:"column:status_code" json:"status_code"`     // 协议状态码。
	Result       int32     `gorm:"column:result" json:"result"`               // 审计结果。
	ReasonCode   string    `gorm:"column:reason_code" json:"reason_code"`     // 原因编码。
	Reason       string    `gorm:"column:reason" json:"reason"`               // 原因说明。
	LatencyMs    int32     `gorm:"column:latency_ms" json:"latency_ms"`       // 耗时毫秒数。
	RequestSize  int32     `gorm:"column:request_size" json:"request_size"`   // 请求大小。
	ResponseSize int32     `gorm:"column:response_size" json:"response_size"` // 响应大小。
	ClientIP     string    `gorm:"column:client_ip" json:"client_ip"`         // 客户端地址。
	UserAgent    string    `gorm:"column:user_agent" json:"user_agent"`       // 用户代理。
	OccurredAt   time.Time `gorm:"column:occurred_at" json:"occurred_at"`     // 事件发生时间。
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`       // 日志创建时间。
}

// TableName 返回 API 访问日志表名。
func (*BaseAPILog) TableName() string { return TableNameBaseAPILog }
