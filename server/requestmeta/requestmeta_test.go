package requestmeta

import (
	"context"
	"testing"

	"github.com/liujitcn/kratos-kit/server/http/middleware/requestid"
	"go.opentelemetry.io/otel/trace"
)

// TestRequestIDFromHTTPContext 验证请求编号优先复用 HTTP 服务端上下文。
func TestRequestIDFromHTTPContext(t *testing.T) {
	ctx := requestid.WithRequestID(context.Background(), "request-1")
	if requestID := RequestID(ctx); requestID != "request-1" {
		t.Fatalf("unexpected request id: %s", requestID)
	}
}

// TestTraceIDFromSpanContext 验证链路编号优先使用 OpenTelemetry 标准值。
func TestTraceIDFromSpanContext(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: trace.SpanID{1}, TraceFlags: trace.FlagsSampled})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)
	if actual := TraceID(ctx); actual != traceID.String() {
		t.Fatalf("unexpected trace id: %s", actual)
	}
}
