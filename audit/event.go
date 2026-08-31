// Package audit 提供 Core 到业务模块的审计事件扩展点。
package audit

import (
	"time"

	"github.com/liujitcn/kratos-kit/transport/queue"
)

// EventStream 是 Core 审计事件使用的队列流名称。
const EventStream queue.Stream = "base.audit.event"

// Event 描述一次跨传输层的访问审计事实。
type Event struct {
	// Kind 是审计事件分类，例如 api 或 policy_evaluation。
	Kind string
	// RequestID 是请求链路中的请求编号。
	RequestID string
	// TraceID 是分布式链路追踪编号。
	TraceID string
	// RequestTime 是事件对应请求的开始时间。
	RequestTime time.Time
	// Method 是请求使用的协议方法或 HTTP 方法。
	Method string
	// Operation 是 Kratos 服务操作标识。
	Operation string
	// Path 是 HTTP 路径模板或请求路径。
	Path string
	// Referer 是请求来源页面。
	Referer string
	// RequestURI 是请求的完整 URI。
	RequestURI string
	// RequestBody 是脱敏后的请求体摘要。
	RequestBody string
	// RequestHeader 是脱敏后的请求头 JSON。
	RequestHeader string
	// Response 是脱敏后的响应内容或摘要。
	Response string
	// CostTime 是请求处理耗时，单位为毫秒。
	CostTime int64
	// IsSuccess 表示请求或决策是否成功。
	IsSuccess bool
	// StatusCode 是对外返回的协议状态码。
	StatusCode int32
	// Reason 是脱敏后的失败原因或处理说明。
	Reason string
	// ReasonCode 是稳定的错误原因编码。
	ReasonCode string
	// Location 是根据客户端地址解析出的地理位置。
	Location string
	// TenantID 是当前租户编号。
	TenantID int64
	// TenantCode 是当前租户编码。
	TenantCode string
	// UserID 是当前操作者用户编号。
	UserID int64
	// UserName 是当前操作者账号名。
	UserName string
	// ClientIP 是客户端 IP 地址。
	ClientIP string
	// UserAgent 是客户端 User-Agent 信息。
	UserAgent string
	// BrowserName 是客户端浏览器名称。
	BrowserName string
	// BrowserVersion 是客户端浏览器版本。
	BrowserVersion string
	// ClientID 是客户端设备或应用标识。
	ClientID string
	// ClientName 是客户端设备或应用名称。
	ClientName string
	// LoginType 是登录事件类型枚举值。
	LoginType int32
	// DeviceID 是登录设备标识。
	DeviceID string
	// OsName 是客户端操作系统名称。
	OsName string
	// OsVersion 是客户端操作系统版本。
	OsVersion string
	// Engine 是策略评估使用的授权引擎名称。
	Engine string
	// EvaluationType 是策略评估类型枚举值。
	EvaluationType int32
	// Resource 是策略评估所针对的资源。
	Resource string
	// Action 是策略评估使用的动作名称。
	Action string
	// Decision 是策略评估决策枚举值。
	Decision int32
	// DurationMs 是策略评估耗时，单位为毫秒。
	DurationMs int32
	// Project 是策略评估涉及的项目范围。
	Project string
	// CandidateCount 是策略评估候选策略数量。
	CandidateCount int32
	// MatchedCount 是策略评估命中策略数量。
	MatchedCount int32
	// ActionCode 是业务操作或权限动作枚举值。
	ActionCode int32
	// RoleID 是策略评估时的角色编号。
	RoleID int64
	// RoleCode 是策略评估时的角色编码。
	RoleCode string
	// InputHash 是策略输入摘要。
	InputHash string
	// ResourceType 是业务资源类型，例如 user 或 role。
	ResourceType string
	// ResourceID 是被操作资源编号。
	ResourceID string
	// ResourceName 是被操作资源名称。
	ResourceName string
	// ChangedFields 是变更字段 JSON 数组。
	ChangedFields string
	// BeforeData 是变更前数据 JSON 快照。
	BeforeData string
	// AfterData 是变更后数据 JSON 快照。
	AfterData string
	// AccessType 是数据访问类型枚举值。
	AccessType int32
	// DataSource 是数据源名称。
	DataSource string
	// TableName 是访问的数据表名称。
	TableName string
	// FieldScope 是访问字段范围 JSON。
	FieldScope string
	// DataScope 是数据权限范围描述。
	DataScope string
	// AffectedRows 是受影响数据行数。
	AffectedRows int32
	// Sensitive 表示是否访问敏感数据。
	Sensitive int32
	// SQLText 是脱敏后的 SQL 文本。
	SQLText string
	// SQLDigest 是 SQL 指纹摘要。
	SQLDigest string
	// TargetType 是权限目标类型枚举值。
	TargetType int32
	// TargetID 是权限目标编号。
	TargetID string
	// TargetName 是权限目标名称。
	TargetName string
	// OldValue 是权限变更前 JSON 值。
	OldValue string
	// NewValue 是权限变更后 JSON 值。
	NewValue string
}
