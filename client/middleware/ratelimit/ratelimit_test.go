package ratelimit

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testLimiter struct {
	allowed bool
}

// Allow 返回测试配置的立即放行结果。
func (limiter *testLimiter) Allow() (bool, error) {
	return limiter.allowed, nil
}

// Wait 模拟等待模式的放行结果。
func (limiter *testLimiter) Wait(context.Context) error {
	if limiter.allowed {
		return nil
	}
	return context.DeadlineExceeded
}

// TestUnaryClientInterceptorRejectsRequest 验证无可用令牌时一元请求会被拒绝。
func TestUnaryClientInterceptorRejectsRequest(t *testing.T) {
	interceptor := UnaryClientInterceptor(&testLimiter{})
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		t.Fatal("invoker should not be called")
		return nil
	}
	err := interceptor(context.Background(), "/test.Service/GetValue", nil, nil, nil, invoker)
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("error code = %s, want %s", status.Code(err), codes.ResourceExhausted)
	}
}

// TestStreamClientInterceptorAllowsStream 验证有可用令牌时流式请求可以建立。
func TestStreamClientInterceptorAllowsStream(t *testing.T) {
	interceptor := StreamClientInterceptor(&testLimiter{allowed: true})
	expected := &testClientStream{}
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		return expected, nil
	}
	stream, err := interceptor(context.Background(), new(grpc.StreamDesc), nil, "/test.Service/Watch", streamer)
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if stream != expected {
		t.Fatalf("stream = %T, want original stream", stream)
	}
}

type testClientStream struct {
	grpc.ClientStream
}
