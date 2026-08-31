package audit

import (
	"context"
	"testing"
)

type testEmitter struct {
	count int
}

// Emit 记录测试接收器收到的审计事件数量。
func (e *testEmitter) Emit(context.Context, Event) error {
	e.count++
	return nil
}

// TestEmitUsesRegisteredEmitter 验证审计事件会交给已注册的接收器。
func TestEmitUsesRegisteredEmitter(t *testing.T) {
	emitter := &testEmitter{}
	SetEmitter(emitter)
	t.Cleanup(func() { SetEmitter(nil) })
	var err error
	err = Emit(context.Background(), Event{Operation: "/test"})
	if err != nil {
		t.Fatal(err)
	}
	if emitter.count != 1 {
		t.Fatalf("expected one event, got %d", emitter.count)
	}
}
