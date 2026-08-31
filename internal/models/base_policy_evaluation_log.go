package models

import "time"

// TableNameBasePolicyEvaluationLog 是策略评估日志表名。
const TableNameBasePolicyEvaluationLog = "base_policy_evaluation_log"

// BasePolicyEvaluationLog 保存 Core 写入的策略评估日志字段。
// 完整表结构和自动迁移模型由宿主模块提供。
type BasePolicyEvaluationLog struct {
	ID             int64     `gorm:"column:id" json:"id"`                           // 策略评估日志编号。
	TenantID       int64     `gorm:"column:tenant_id" json:"tenant_id"`             // 租户编号。
	TenantCode     string    `gorm:"column:tenant_code" json:"tenant_code"`         // 租户编码。
	UserID         int64     `gorm:"column:user_id" json:"user_id"`                 // 用户编号。
	UserName       string    `gorm:"column:user_name" json:"user_name"`             // 用户名称。
	RoleID         int64     `gorm:"column:role_id" json:"role_id"`                 // 角色编号。
	RoleCode       string    `gorm:"column:role_code" json:"role_code"`             // 角色编码。
	RequestID      string    `gorm:"column:request_id" json:"request_id"`           // 请求编号。
	TraceID        string    `gorm:"column:trace_id" json:"trace_id"`               // 链路追踪编号。
	ClientIP       string    `gorm:"column:client_ip" json:"client_ip"`             // 客户端地址。
	Engine         string    `gorm:"column:engine" json:"engine"`                   // 鉴权引擎。
	EvaluationType int32     `gorm:"column:evaluation_type" json:"evaluation_type"` // 评估类型。
	Resource       string    `gorm:"column:resource" json:"resource"`               // 评估资源。
	Action         string    `gorm:"column:action" json:"action"`                   // 评估动作。
	Project        string    `gorm:"column:project" json:"project"`                 // 项目范围。
	Decision       int32     `gorm:"column:decision" json:"decision"`               // 决策结果。
	ReasonCode     string    `gorm:"column:reason_code" json:"reason_code"`         // 原因编码。
	Reason         string    `gorm:"column:reason" json:"reason"`                   // 原因说明。
	DurationMs     int32     `gorm:"column:duration_ms" json:"duration_ms"`         // 评估耗时毫秒数。
	CandidateCount int32     `gorm:"column:candidate_count" json:"candidate_count"` // 候选数量。
	MatchedCount   int32     `gorm:"column:matched_count" json:"matched_count"`     // 命中数量。
	InputHash      string    `gorm:"column:input_hash" json:"input_hash"`           // 输入摘要。
	OccurredAt     time.Time `gorm:"column:occurred_at" json:"occurred_at"`         // 事件发生时间。
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`           // 日志创建时间。
}

// TableName 返回策略评估日志表名。
func (*BasePolicyEvaluationLog) TableName() string { return TableNameBasePolicyEvaluationLog }
