package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/server/http/middleware/requestid"
)

type logTestAuthorizer struct{}

// Name 返回测试授权引擎名称。
func (*logTestAuthorizer) Name() string { return "test" }

// ProjectsAuthorized 返回测试项目集合。
func (*logTestAuthorizer) ProjectsAuthorized(_ context.Context, _ engine.Subjects, _ engine.Action, _ engine.Resource, projects engine.Projects) (engine.Projects, error) {
	return projects, nil
}

// FilterAuthorizedPairs 返回测试资源动作对。
func (*logTestAuthorizer) FilterAuthorizedPairs(_ context.Context, _ engine.Subjects, pairs engine.Pairs) (engine.Pairs, error) {
	return pairs, nil
}

// FilterAuthorizedProjects 返回空测试项目集合。
func (*logTestAuthorizer) FilterAuthorizedProjects(context.Context, engine.Subjects) (engine.Projects, error) {
	return nil, nil
}

// IsAuthorized 模拟一次短暂的真实策略评估。
func (*logTestAuthorizer) IsAuthorized(context.Context, engine.Subject, engine.Action, engine.Resource, engine.Project) (bool, error) {
	time.Sleep(10 * time.Millisecond)
	return true, nil
}

type logCaptureEmitter struct {
	event biz.LogEvent
	err   error
}

// Emit 捕获测试策略事件并返回配置的投递错误。
func (e *logCaptureEmitter) Emit(_ context.Context, event biz.LogEvent) error {
	e.event = event
	return e.err
}

// TestLoggedAuthorizerMeasuresOnlyPolicyEvaluation 验证策略耗时不包含后续业务处理时间。
func TestLoggedAuthorizerMeasuresOnlyPolicyEvaluation(t *testing.T) {
	emitter := &logCaptureEmitter{err: errors.New("queue unavailable")}
	biz.SetLogEmitter(emitter)
	t.Cleanup(func() { biz.SetLogEmitter(nil) })
	tenant := engine.Tenant("default")
	subject := engine.Subject("admin")
	action := engine.Action("POST")
	resource := engine.Resource("/system.admin.v1.BaseUserService/UpdateBaseUser")
	claims := &engine.AuthClaims{Tenant: &tenant, Subject: &subject, Action: &action, Resource: &resource}
	ctx := engine.ContextWithAuthClaims(context.Background(), claims)
	ctx = requestid.WithRequestID(ctx, "request-1")
	handler := auditedAuthzMiddleware(&logTestAuthorizer{})(func(context.Context, interface{}) (interface{}, error) {
		time.Sleep(80 * time.Millisecond)
		return "business-result", nil
	})
	reply, err := handler(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reply != "business-result" {
		t.Fatalf("unexpected business result: %v", reply)
	}
	if emitter.event.RequestID != "request-1" {
		t.Fatalf("unexpected request id: %s", emitter.event.RequestID)
	}
	if emitter.event.DurationMs < 10 || emitter.event.DurationMs >= 50 {
		t.Fatalf("unexpected policy duration: %d", emitter.event.DurationMs)
	}
}
