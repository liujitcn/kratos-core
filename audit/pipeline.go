package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/kratos-core/data"
	"github.com/liujitcn/kratos-core/internal/models"
	"github.com/liujitcn/kratos-kit/database/gorm"
	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

const (
	auditResultSuccess int32 = 1
	auditResultFailure int32 = 2
	auditResultError   int32 = 3
)

type apiLogWriter interface {
	Create(context.Context, *models.BaseAPILog) error
}

type policyEvaluationLogWriter interface {
	Create(context.Context, *models.BasePolicyEvaluationLog) error
}

// Pipeline 负责 Core 审计事件的队列投递和异步入库。
type Pipeline struct {
	queue                     kitQueue.Queue
	apiLogWriter              apiLogWriter
	policyEvaluationLogWriter policyEvaluationLogWriter
}

var _ Emitter = (*Pipeline)(nil)

// NewPipeline 创建 Core 审计流水线并注册为进程级事件接收器。
func NewPipeline(
	queue kitQueue.Queue,
	apiLogRepository *data.BaseAPILogRepository,
	policyEvaluationLogRepository *data.BasePolicyEvaluationLogRepository,
) (*Pipeline, func()) {
	pipeline := &Pipeline{
		queue:                     queue,
		apiLogWriter:              apiLogRepository,
		policyEvaluationLogWriter: policyEvaluationLogRepository,
	}
	SetEmitter(pipeline)
	return pipeline, func() {
		SetEmitter(nil)
	}
}

// Emit 将 Core 审计事件投递到异步入库队列。
func (p *Pipeline) Emit(_ context.Context, event Event) error {
	event = normalizeEvent(event)
	var rawBody []byte
	var err error
	rawBody, err = json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化 Core 审计事件失败: %w", err)
	}
	err = p.queue.Append(string(EventStream), queueData.Message{Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		return fmt.Errorf("投递 Core 审计事件失败: %w", err)
	}
	return nil
}

// Consume 消费 Core 审计事件并写入对应日志表。
func (p *Pipeline) Consume(message queueData.Message) error {
	rawValue, ok := message.Values["data"].(string)
	if !ok || rawValue == "" {
		return fmt.Errorf("Core 审计事件消息体为空")
	}
	var event Event
	var err error
	err = json.Unmarshal([]byte(rawValue), &event)
	if err != nil {
		return fmt.Errorf("解析 Core 审计事件失败: %w", err)
	}
	event = normalizeEvent(event)
	return p.persist(context.Background(), event)
}

// persist 根据事件类型写入 Core 负责的日志表。
func (p *Pipeline) persist(ctx context.Context, event Event) error {
	switch event.Kind {
	case "api":
		return p.apiLogWriter.Create(ctx, apiLogFromEvent(event))
	case "policy_evaluation":
		return p.policyEvaluationLogWriter.Create(ctx, policyEvaluationLogFromEvent(event))
	default:
		return fmt.Errorf("未知 Core 审计事件类型: %s", event.Kind)
	}
}

// normalizeEvent 补齐 Core 审计事件的通用默认值。
func normalizeEvent(event Event) Event {
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

// apiLogFromEvent 将 Core 审计事件转换为 API 访问日志模型。
func apiLogFromEvent(event Event) *models.BaseAPILog {
	return &models.BaseAPILog{
		TenantID: event.TenantID, TenantCode: event.TenantCode, UserID: event.UserID, UserName: event.UserName,
		RequestID: event.RequestID, TraceID: event.TraceID, ServiceName: serviceName(event.Operation),
		Operation: event.Operation, Method: event.Method, Path: event.Path, StatusCode: event.StatusCode,
		Result: auditResult(event.IsSuccess, event.StatusCode), ReasonCode: event.ReasonCode, Reason: event.Reason,
		LatencyMs: int32(event.CostTime), ClientIP: event.ClientIP, UserAgent: event.UserAgent,
		OccurredAt: event.RequestTime, CreatedAt: time.Now(),
	}
}

// policyEvaluationLogFromEvent 将 Core 审计事件转换为策略评估日志模型。
func policyEvaluationLogFromEvent(event Event) *models.BasePolicyEvaluationLog {
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

// auditResult 根据请求结果生成稳定的审计结果枚举值。
func auditResult(success bool, statusCode int32) int32 {
	if success {
		return auditResultSuccess
	}
	if statusCode >= http.StatusInternalServerError {
		return auditResultError
	}
	return auditResultFailure
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
