package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/liujitcn/kratos-core/internal/models"
	"github.com/liujitcn/kratos-kit/queue/data"
)

type testQueue struct {
	stream  string
	message data.Message
}

// Append 记录测试期间投递的队列消息。
func (q *testQueue) Append(stream string, message data.Message) error {
	q.stream = stream
	q.message = message
	return nil
}

// Register 忽略测试不需要的消费者注册。
func (*testQueue) Register(string, data.ConsumerFunc) {}

// Run 忽略测试不需要的队列启动。
func (*testQueue) Run() {}

// Shutdown 忽略测试不需要的队列停止。
func (*testQueue) Shutdown() {}

type testAPILogWriter struct {
	item *models.BaseAPILog
}

// Create 记录测试期间写入的 API 访问日志。
func (w *testAPILogWriter) Create(_ context.Context, item *models.BaseAPILog) error {
	w.item = item
	return nil
}

type testPolicyEvaluationLogWriter struct {
	item *models.BasePolicyEvaluationLog
}

// Create 记录测试期间写入的策略评估日志。
func (w *testPolicyEvaluationLogWriter) Create(_ context.Context, item *models.BasePolicyEvaluationLog) error {
	w.item = item
	return nil
}

// TestPipelineEmitsToCoreStream 验证 Core 流水线会补齐默认值并投递审计事件。
func TestPipelineEmitsToCoreStream(t *testing.T) {
	queue := &testQueue{}
	pipeline := &Pipeline{queue: queue}
	var err error
	err = pipeline.Emit(context.Background(), Event{Operation: "/base.v1.TestService/Get"})
	if err != nil {
		t.Fatal(err)
	}
	if queue.stream != string(EventStream) {
		t.Fatalf("unexpected stream: %s", queue.stream)
	}
	rawBody, ok := queue.message.Values["data"].(string)
	if !ok {
		t.Fatal("expected string queue payload")
	}
	var event Event
	err = json.Unmarshal([]byte(rawBody), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != "api" || event.RequestID == "" || event.RequestTime.IsZero() {
		t.Fatalf("event defaults not applied: %+v", event)
	}
}

// TestPipelineConsumesAPILog 验证 Core 流水线消费 API 事件并写入 API 日志模型。
func TestPipelineConsumesAPILog(t *testing.T) {
	apiWriter := &testAPILogWriter{}
	policyWriter := &testPolicyEvaluationLogWriter{}
	pipeline := &Pipeline{apiLogWriter: apiWriter, policyEvaluationLogWriter: policyWriter}
	event := Event{
		Kind: "api", RequestID: "request", RequestTime: time.Now(), Operation: "/base.v1.TestService/Get",
		Method: "GET", Path: "/test", StatusCode: 200, IsSuccess: true,
	}
	rawBody, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	err = pipeline.Consume(data.Message{Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		t.Fatal(err)
	}
	if apiWriter.item == nil {
		t.Fatal("expected API log write")
	}
	if apiWriter.item.ServiceName != "TestService" || apiWriter.item.Result != auditResultSuccess {
		t.Fatalf("unexpected API log: %+v", apiWriter.item)
	}
	if policyWriter.item != nil {
		t.Fatal("unexpected policy log write")
	}
}

// TestPipelineConsumesPolicyEvaluationLog 验证 Core 流水线消费策略事件并写入策略评估日志模型。
func TestPipelineConsumesPolicyEvaluationLog(t *testing.T) {
	apiWriter := &testAPILogWriter{}
	policyWriter := &testPolicyEvaluationLogWriter{}
	pipeline := &Pipeline{apiLogWriter: apiWriter, policyEvaluationLogWriter: policyWriter}
	event := Event{
		Kind: "policy_evaluation", RequestID: "request", RequestTime: time.Now(),
		Engine: "casbin", Resource: "/base.v1.TestService/Get", Action: "GET", Decision: 1,
	}
	rawBody, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	err = pipeline.Consume(data.Message{Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		t.Fatal(err)
	}
	if policyWriter.item == nil {
		t.Fatal("expected policy evaluation log write")
	}
	if policyWriter.item.Engine != "casbin" || policyWriter.item.Decision != 1 {
		t.Fatalf("unexpected policy evaluation log: %+v", policyWriter.item)
	}
	if apiWriter.item != nil {
		t.Fatal("unexpected API log write")
	}
}
