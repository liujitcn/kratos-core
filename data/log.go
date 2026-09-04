package data

import (
	"context"
	"time"
)

// APILogRecord 表示 API 访问日志所需的字段。
type APILogRecord struct {
	// ID 是日志编号。
	ID int64
	// TenantID 是租户编号。
	TenantID int64
	// TenantCode 是租户编码。
	TenantCode string
	// UserID 是用户编号。
	UserID int64
	// UserName 是用户名称。
	UserName string
	// RequestID 是请求编号。
	RequestID string
	// TraceID 是链路追踪编号。
	TraceID string
	// ServiceName 是服务名称。
	ServiceName string
	// Operation 是操作标识。
	Operation string
	// Method 是请求方法。
	Method string
	// Path 是请求路径。
	Path string
	// StatusCode 是协议状态码。
	StatusCode int32
	// Result 是日志结果。
	Result int32
	// ReasonCode 是原因编码。
	ReasonCode string
	// Reason 是原因说明。
	Reason string
	// LatencyMs 是请求耗时，单位为毫秒。
	LatencyMs int32
	// ClientIP 是客户端地址。
	ClientIP string
	// UserAgent 是用户代理。
	UserAgent string
	// OccurredAt 是事件发生时间。
	OccurredAt time.Time
	// CreatedAt 是日志创建时间。
	CreatedAt time.Time
}

// PolicyEvaluationLogRecord 表示策略评估日志所需的字段。
type PolicyEvaluationLogRecord struct {
	// ID 是日志编号。
	ID int64
	// TenantID 是租户编号。
	TenantID int64
	// TenantCode 是租户编码。
	TenantCode string
	// UserID 是用户编号。
	UserID int64
	// UserName 是用户名称。
	UserName string
	// RoleID 是角色编号。
	RoleID int64
	// RoleCode 是角色编码。
	RoleCode string
	// RequestID 是请求编号。
	RequestID string
	// TraceID 是链路追踪编号。
	TraceID string
	// ClientIP 是客户端地址。
	ClientIP string
	// Engine 是鉴权引擎。
	Engine string
	// EvaluationType 是评估类型。
	EvaluationType int32
	// Resource 是评估资源。
	Resource string
	// Action 是评估动作。
	Action string
	// Project 是项目范围。
	Project string
	// Decision 是决策结果。
	Decision int32
	// ReasonCode 是原因编码。
	ReasonCode string
	// Reason 是原因说明。
	Reason string
	// DurationMs 是评估耗时，单位为毫秒。
	DurationMs int32
	// CandidateCount 是候选策略数量。
	CandidateCount int32
	// MatchedCount 是命中策略数量。
	MatchedCount int32
	// InputHash 是输入摘要。
	InputHash string
	// OccurredAt 是事件发生时间。
	OccurredAt time.Time
	// CreatedAt 是日志创建时间。
	CreatedAt time.Time
}

// LogStore 提供 Core 审计日志的宿主持久化能力。
type LogStore interface {
	// CreateAPI 创建 API 访问日志。
	CreateAPI(context.Context, APILogRecord) error
	// ExistsAPI 判断指定 API 日志是否已经写入。
	ExistsAPI(context.Context, int64) (bool, error)
	// CreatePolicyEvaluation 创建策略评估日志。
	CreatePolicyEvaluation(context.Context, PolicyEvaluationLogRecord) error
	// ExistsPolicyEvaluation 判断指定策略评估日志是否已经写入。
	ExistsPolicyEvaluation(context.Context, int64) (bool, error)
}
