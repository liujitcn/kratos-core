package retry

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	coreRetry "github.com/liujitcn/kratos-kit/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestUnaryClientInterceptor 验证幂等方法会按配置重试并返回最终结果。
func TestUnaryClientInterceptor(t *testing.T) {
	retrier := coreRetry.New(
		coreRetry.WithMaxAttempts(3),
		coreRetry.WithBackoff(coreRetry.FixedBackoff(time.Nanosecond)),
	)
	interceptor := UnaryClientInterceptor(retrier)
	var calls atomic.Int32
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		if calls.Add(1) < 3 {
			return status.Error(codes.Unavailable, "temporary")
		}
		return nil
	}
	err := interceptor(context.Background(), "/test.Service/GetValue", nil, nil, nil, invoker)
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

// TestUnaryClientInterceptorSkipsNonIdempotentMethod 验证非幂等方法不会自动重试。
func TestUnaryClientInterceptorSkipsNonIdempotentMethod(t *testing.T) {
	retrier := coreRetry.New(coreRetry.WithMaxAttempts(3))
	interceptor := UnaryClientInterceptor(retrier)
	var calls atomic.Int32
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		calls.Add(1)
		return status.Error(codes.Unavailable, "temporary")
	}
	err := interceptor(context.Background(), "/test.Service/CreateValue", nil, nil, nil, invoker)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("error code = %s, want %s", status.Code(err), codes.Unavailable)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}
