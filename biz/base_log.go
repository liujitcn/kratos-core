package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/internal/models"
	"github.com/liujitcn/kratos-kit/database/gorm"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

const (
	logResultSuccess    int32 = 1
	logResultFailure    int32 = 2
	logResultError      int32 = 3
	logBufferSize             = 2048
	messageSequenceBits       = 20
)

// LogEventStream 是 Core 日志事件使用的队列流名称。
const LogEventStream queueTransport.Stream = "base.log.event"

// LogEvent 描述一次跨传输层的日志事实。
type LogEvent struct {
	// Kind 是日志事件分类，例如 api 或 policy_evaluation。
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

// LogEmitter 接收 Core 产生的日志事件。
type LogEmitter interface {
	Emit(context.Context, LogEvent) error
}

// LogSink 消费 Core 日志事件并写入对应日志表。
type LogSink interface {
	Consume(queueData.Message) error
}

var logEmitterState struct {
	sync.RWMutex
	emitter LogEmitter
}

// SetLogEmitter 设置当前进程的日志事件接收器。
func SetLogEmitter(value LogEmitter) {
	logEmitterState.Lock()
	logEmitterState.emitter = value
	logEmitterState.Unlock()
}

// EmitLog 将日志事件交给当前接收器；未配置接收器时静默忽略。
func EmitLog(ctx context.Context, event LogEvent) error {
	logEmitterState.RLock()
	value := logEmitterState.emitter
	logEmitterState.RUnlock()
	if value == nil {
		return nil
	}
	return value.Emit(ctx, event)
}

type apiLogWriter interface {
	Create(context.Context, *models.BaseAPILog) error
	FindByID(context.Context, int64) (*models.BaseAPILog, error)
}

type policyEvaluationLogWriter interface {
	Create(context.Context, *models.BasePolicyEvaluationLog) error
	FindByID(context.Context, int64) (*models.BasePolicyEvaluationLog, error)
}

// LogPipeline 负责 Core 日志事件的队列投递和异步入库。
type LogPipeline struct {
	queue                     kitQueue.Queue
	apiLogWriter              apiLogWriter
	policyEvaluationLogWriter policyEvaluationLogWriter
	events                    chan LogEvent
	worker                    sync.WaitGroup
	stateMu                   sync.RWMutex
	closed                    bool
}

var _ LogEmitter = (*LogPipeline)(nil)
var _ LogSink = (*LogPipeline)(nil)

// NewLogPipeline 创建 Core 日志流水线并注册为进程级事件接收器。
func NewLogPipeline(
	queue kitQueue.Queue,
	apiLogRepository *data.BaseAPILogRepository,
	policyEvaluationLogRepository *data.BasePolicyEvaluationLogRepository,
) (*LogPipeline, func()) {
	pipeline := newLogPipeline(queue, apiLogRepository, policyEvaluationLogRepository)
	SetLogEmitter(pipeline)
	return pipeline, func() {
		SetLogEmitter(nil)
		pipeline.close()
	}
}

// Emit 将 Core 日志事件非阻塞投递到进程内缓冲。
func (p *LogPipeline) Emit(_ context.Context, event LogEvent) error {
	event = normalizeLogEvent(event)
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	if p.closed {
		return fmt.Errorf("Core 日志事件缓冲已关闭")
	}
	select {
	case p.events <- event:
		return nil
	default:
		return fmt.Errorf("Core 日志事件缓冲已满")
	}
}

// run 在后台序列化并投递 Core 日志事件，避免队列 IO 阻塞请求线程。
func (p *LogPipeline) run() {
	defer p.worker.Done()
	for event := range p.events {
		rawBody, err := json.Marshal(event)
		if err != nil {
			log.Error("序列化 Core 日志事件失败", "error", err, "kind", event.Kind)
			continue
		}
		err = p.queue.Append(string(LogEventStream), queueData.Message{Values: map[string]interface{}{"data": string(rawBody)}})
		if err != nil {
			log.Error("投递 Core 日志事件失败", "error", err, "kind", event.Kind)
		}
	}
}

// close 关闭事件缓冲并等待已接收的日志事件完成队列投递。
func (p *LogPipeline) close() {
	p.stateMu.Lock()
	if !p.closed {
		p.closed = true
		close(p.events)
	}
	p.stateMu.Unlock()
	p.worker.Wait()
}

// Consume 消费 Core 日志事件并写入对应日志表。
func (p *LogPipeline) Consume(message queueData.Message) error {
	rawValue, ok := message.Values["data"].(string)
	if !ok || rawValue == "" {
		return fmt.Errorf("Core 日志事件消息体为空")
	}
	var event LogEvent
	var err error
	err = json.Unmarshal([]byte(rawValue), &event)
	if err != nil {
		return fmt.Errorf("解析 Core 日志事件失败: %w", err)
	}
	event = normalizeLogEvent(event)
	return p.persist(context.Background(), event, LogMessagePrimaryKey(message.ID))
}

// persist 根据事件类型写入 Core 负责的日志表。
func (p *LogPipeline) persist(ctx context.Context, event LogEvent, recordID int64) error {
	switch event.Kind {
	case "api":
		item := apiLogFromEvent(event)
		item.ID = recordID
		err := p.apiLogWriter.Create(ctx, item)
		if err == nil || recordID == 0 {
			return err
		}
		var existing *models.BaseAPILog
		var lookupErr error
		existing, lookupErr = p.apiLogWriter.FindByID(ctx, recordID)
		if lookupErr == nil && existing != nil {
			return nil
		}
		return err
	case "policy_evaluation":
		item := policyEvaluationLogFromEvent(event)
		item.ID = recordID
		err := p.policyEvaluationLogWriter.Create(ctx, item)
		if err == nil || recordID == 0 {
			return err
		}
		var existing *models.BasePolicyEvaluationLog
		var lookupErr error
		existing, lookupErr = p.policyEvaluationLogWriter.FindByID(ctx, recordID)
		if lookupErr == nil && existing != nil {
			return nil
		}
		return err
	default:
		return fmt.Errorf("未知 Core 日志事件类型: %s", event.Kind)
	}
}

// normalizeLogEvent 补齐 Core 日志事件的通用默认值。
func normalizeLogEvent(event LogEvent) LogEvent {
	if event.Kind == "" {
		event.Kind = "api"
	}
	if event.RequestTime.IsZero() {
		event.RequestTime = time.Now()
	}
	if event.RequestID == "" {
		event.RequestID = id.NewGUIDv4NoHyphen()
	}
	if event.TenantCode == "" {
		event.TenantCode = gorm.DefaultTenantCode
	}
	return event
}

// apiLogFromEvent 将 Core 日志事件转换为 API 访问日志模型。
func apiLogFromEvent(event LogEvent) *models.BaseAPILog {
	return &models.BaseAPILog{
		TenantID: event.TenantID, TenantCode: event.TenantCode, UserID: event.UserID, UserName: event.UserName,
		RequestID: event.RequestID, TraceID: event.TraceID, ServiceName: serviceName(event.Operation),
		Operation: event.Operation, Method: event.Method, Path: event.Path, StatusCode: event.StatusCode,
		Result: logResult(event.IsSuccess, event.StatusCode), ReasonCode: event.ReasonCode, Reason: event.Reason,
		LatencyMs: int32(event.CostTime), ClientIP: event.ClientIP, UserAgent: event.UserAgent,
		OccurredAt: event.RequestTime, CreatedAt: time.Now(),
	}
}

// policyEvaluationLogFromEvent 将 Core 日志事件转换为策略评估日志模型。
func policyEvaluationLogFromEvent(event LogEvent) *models.BasePolicyEvaluationLog {
	return &models.BasePolicyEvaluationLog{
		TenantID: event.TenantID, TenantCode: event.TenantCode, UserID: event.UserID, UserName: event.UserName,
		RoleID: event.RoleID, RoleCode: event.RoleCode, RequestID: event.RequestID, TraceID: event.TraceID,
		ClientIP: event.ClientIP, Engine: event.Engine, EvaluationType: event.EvaluationType,
		Resource: event.Resource, Action: event.Action, Project: event.Project, Decision: event.Decision,
		ReasonCode: event.ReasonCode, Reason: event.Reason, DurationMs: event.DurationMs,
		CandidateCount: event.CandidateCount, MatchedCount: event.MatchedCount, InputHash: event.InputHash,
		OccurredAt: event.RequestTime, CreatedAt: time.Now(),
	}
}

// logResult 根据请求结果生成稳定的日志结果枚举值。
func logResult(success bool, statusCode int32) int32 {
	if success {
		return logResultSuccess
	}
	if statusCode >= http.StatusInternalServerError {
		return logResultError
	}
	return logResultFailure
}

// serviceName 从 Kratos operation 中提取服务名称。
func serviceName(operation string) string {
	parts := strings.Split(strings.TrimPrefix(operation, "/"), "/")
	if len(parts) != 2 {
		return operation
	}
	serviceParts := strings.Split(parts[0], ".")
	return serviceParts[len(serviceParts)-1]
}

// newLogPipeline 创建并启动 Core 日志事件后台投递器。
func newLogPipeline(queue kitQueue.Queue, apiWriter apiLogWriter, policyWriter policyEvaluationLogWriter) *LogPipeline {
	pipeline := &LogPipeline{
		queue:                     queue,
		apiLogWriter:              apiWriter,
		policyEvaluationLogWriter: policyWriter,
		events:                    make(chan LogEvent, logBufferSize),
	}
	pipeline.worker.Add(1)
	go pipeline.run()
	return pipeline
}

// LogMessagePrimaryKey 将稳定队列消息编号映射为日志表幂等主键。
func LogMessagePrimaryKey(messageID string) int64 {
	if messageID == "" {
		return 0
	}
	parts := strings.SplitN(messageID, "-", 2)
	if len(parts) == 2 {
		timestamp, err := strconv.ParseInt(parts[0], 10, 64)
		if err == nil && timestamp > 0 && timestamp <= math.MaxInt64>>messageSequenceBits {
			var sequence int64
			sequence, err = strconv.ParseInt(parts[1], 10, 64)
			if err == nil && sequence >= 0 && sequence < 1<<messageSequenceBits {
				return timestamp<<messageSequenceBits | sequence
			}
		}
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(messageID))
	value := int64(hasher.Sum64() & math.MaxInt64)
	if value == 0 {
		return 1
	}
	return value
}
