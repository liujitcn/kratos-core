package biz

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/liujitcn/kratos-core/internal/models"
	"github.com/liujitcn/kratos-kit/queue/data"
)

type testQueue struct {
	stream  string
	message data.Message
	started chan struct{}
	release chan struct{}
}

// Append 记录测试期间投递的队列消息。
func (q *testQueue) Append(stream string, message data.Message) error {
	q.stream = stream
	q.message = message
	if q.started != nil {
		close(q.started)
		<-q.release
	}
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
	if w.item != nil && w.item.ID == item.ID && item.ID != 0 {
		return errors.New("duplicate primary key")
	}
	w.item = item
	return nil
}

// FindByID 返回测试期间已写入的 API 访问日志。
func (w *testAPILogWriter) FindByID(_ context.Context, id int64) (*models.BaseAPILog, error) {
	if w.item != nil && w.item.ID == id {
		return w.item, nil
	}
	return nil, errors.New("not found")
}

type testPolicyEvaluationLogWriter struct {
	item *models.BasePolicyEvaluationLog
}

// Create 记录测试期间写入的策略评估日志。
func (w *testPolicyEvaluationLogWriter) Create(_ context.Context, item *models.BasePolicyEvaluationLog) error {
	if w.item != nil && w.item.ID == item.ID && item.ID != 0 {
		return errors.New("duplicate primary key")
	}
	w.item = item
	return nil
}

// FindByID 返回测试期间已写入的策略评估日志。
func (w *testPolicyEvaluationLogWriter) FindByID(_ context.Context, id int64) (*models.BasePolicyEvaluationLog, error) {
	if w.item != nil && w.item.ID == id {
		return w.item, nil
	}
	return nil, errors.New("not found")
}

// TestLogPipelineEmitsToCoreStream 验证 Core 流水线会补齐默认值并投递日志事件。
func TestLogPipelineEmitsToCoreStream(t *testing.T) {
	queue := &testQueue{}
	pipeline := newLogPipeline(queue, &testAPILogWriter{}, &testPolicyEvaluationLogWriter{})
	var err error
	err = pipeline.Emit(context.Background(), LogEvent{Operation: "/base.v1.TestService/Get"})
	if err != nil {
		t.Fatal(err)
	}
	pipeline.close()
	if queue.stream != string(LogEventStream) {
		t.Fatalf("unexpected stream: %s", queue.stream)
	}
	rawBody, ok := queue.message.Values["data"].(string)
	if !ok {
		t.Fatal("expected string queue payload")
	}
	var event LogEvent
	err = json.Unmarshal([]byte(rawBody), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != "api" || event.RequestID == "" || event.RequestTime.IsZero() {
		t.Fatalf("event defaults not applied: %+v", event)
	}
}

// TestLogPipelineEmitDoesNotWaitForQueueIO 验证 Core 日志投递不会让请求线程等待队列 IO。
func TestLogPipelineEmitDoesNotWaitForQueueIO(t *testing.T) {
	queue := &testQueue{started: make(chan struct{}), release: make(chan struct{})}
	pipeline := newLogPipeline(queue, &testAPILogWriter{}, &testPolicyEvaluationLogWriter{})
	defer func() {
		close(queue.release)
		pipeline.close()
	}()
	startedAt := time.Now()
	err := pipeline.Emit(context.Background(), LogEvent{Operation: "/base.v1.TestService/Get"})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(startedAt) > 100*time.Millisecond {
		t.Fatal("log emit waited for queue IO")
	}
	select {
	case <-queue.started:
	case <-time.After(time.Second):
		t.Fatal("log worker did not dispatch queued event")
	}
}

// TestLogPipelineConsumesAPILog 验证 Core 流水线消费 API 事件并写入 API 日志模型。
func TestLogPipelineConsumesAPILog(t *testing.T) {
	apiWriter := &testAPILogWriter{}
	policyWriter := &testPolicyEvaluationLogWriter{}
	pipeline := newLogPipeline(&testQueue{}, apiWriter, policyWriter)
	defer pipeline.close()
	event := LogEvent{
		Kind: "api", RequestID: "request", RequestTime: time.Now(), Operation: "/base.v1.TestService/Get",
		Method: "GET", Path: "/test", StatusCode: 200, IsSuccess: true,
	}
	rawBody, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	err = pipeline.Consume(data.Message{ID: "1700000000000-1", Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		t.Fatal(err)
	}
	if apiWriter.item == nil {
		t.Fatal("expected API log write")
	}
	if apiWriter.item.ServiceName != "TestService" || apiWriter.item.Result != logResultSuccess {
		t.Fatalf("unexpected API log: %+v", apiWriter.item)
	}
	if policyWriter.item != nil {
		t.Fatal("unexpected policy log write")
	}
	err = pipeline.Consume(data.Message{ID: "1700000000000-1", Values: map[string]interface{}{"data": string(rawBody)}})
	if err != nil {
		t.Fatalf("duplicate delivery must be idempotent: %v", err)
	}
}

// TestLogPipelineConsumesPolicyEvaluationLog 验证 Core 流水线消费策略事件并写入策略评估日志模型。
func TestLogPipelineConsumesPolicyEvaluationLog(t *testing.T) {
	apiWriter := &testAPILogWriter{}
	policyWriter := &testPolicyEvaluationLogWriter{}
	pipeline := newLogPipeline(&testQueue{}, apiWriter, policyWriter)
	defer pipeline.close()
	event := LogEvent{
		Kind: "policy_evaluation", RequestID: "request", RequestTime: time.Now(),
		Engine: "casbin", Resource: "/base.v1.TestService/Get", Action: "GET", Decision: 1,
	}
	rawBody, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	err = pipeline.Consume(data.Message{ID: "1700000000000-2", Values: map[string]interface{}{"data": string(rawBody)}})
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

type testLogEmitter struct {
	count int
}

// Emit 记录测试接收器收到的日志事件数量。
func (e *testLogEmitter) Emit(context.Context, LogEvent) error {
	e.count++
	return nil
}

// TestEmitLogUsesRegisteredEmitter 验证日志事件会交给已注册的接收器。
func TestEmitLogUsesRegisteredEmitter(t *testing.T) {
	emitter := &testLogEmitter{}
	SetLogEmitter(emitter)
	t.Cleanup(func() { SetLogEmitter(nil) })
	var err error
	err = EmitLog(context.Background(), LogEvent{Operation: "/test"})
	if err != nil {
		t.Fatal(err)
	}
	if emitter.count != 1 {
		t.Fatalf("expected one event, got %d", emitter.count)
	}
}

// TestLogMessagePrimaryKeyUsesRedisStreamIdentity 验证 Redis Stream 编号可稳定映射为正整数主键。
func TestLogMessagePrimaryKeyUsesRedisStreamIdentity(t *testing.T) {
	first := LogMessagePrimaryKey("1700000000000-42")
	second := LogMessagePrimaryKey("1700000000000-42")
	if first <= 0 || first != second {
		t.Fatalf("unexpected primary key: %d %d", first, second)
	}
	if first == LogMessagePrimaryKey("1700000000000-43") {
		t.Fatal("different stream messages must not share a primary key")
	}
}

// TestLogMessagePrimaryKeyFallsBackForMemoryQueueID 验证内存队列 UUID 也能稳定生成主键。
func TestLogMessagePrimaryKeyFallsBackForMemoryQueueID(t *testing.T) {
	first := LogMessagePrimaryKey("018f47d2-92f4-7abc-8def-0123456789ab")
	if first <= 0 || first != LogMessagePrimaryKey("018f47d2-92f4-7abc-8def-0123456789ab") {
		t.Fatalf("unexpected fallback primary key: %d", first)
	}
}
