// Package requestmeta 提供服务端请求关联标识的统一读取能力。
package requestmeta

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-kit/server/grpc/middleware/requestid"
	httpRequestID "github.com/liujitcn/kratos-kit/server/http/middleware/requestid"
	"go.opentelemetry.io/otel/trace"
)

// RequestID 从统一请求上下文或传输头读取请求编号。
func RequestID(ctx context.Context) string {
	if requestID := httpRequestID.FromContext(ctx); requestID != "" {
		return requestID
	}
	if requestID := requestid.FromContext(ctx); requestID != "" {
		return requestID
	}
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	for _, name := range []string{"X-Request-ID", "X-Correlation-ID", "X-Fc-Request-Id"} {
		if requestID := serverTransport.RequestHeader().Get(name); requestID != "" {
			return requestID
		}
	}
	return ""
}

// TraceID 从 OpenTelemetry 上下文或 traceparent 请求头读取链路编号。
func TraceID(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		return spanContext.TraceID().String()
	}
	serverTransport, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	traceparent := serverTransport.RequestHeader().Get("traceparent")
	parts := strings.Split(traceparent, "-")
	if len(parts) == 4 && len(parts[1]) == 32 {
		return parts[1]
	}
	return traceparent
}
