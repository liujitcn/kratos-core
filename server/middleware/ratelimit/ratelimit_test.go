package ratelimit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/liujitcn/kratos-kit/cache"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testResolver struct {
	policies []Policy
	err      error
}

// Resolve 返回测试策略集合。
func (resolver testResolver) Resolve(context.Context, string) ([]Policy, error) {
	return resolver.policies, resolver.err
}

type testTransport struct {
	operation string
	headers   testHeader
}

// Kind 返回测试传输类型。
func (tr *testTransport) Kind() transport.Kind { return transport.KindGRPC }

// Endpoint 返回测试传输地址。
func (tr *testTransport) Endpoint() string { return "" }

// Operation 返回测试请求操作名。
func (tr *testTransport) Operation() string { return tr.operation }

// RequestHeader 返回测试请求头。
func (tr *testTransport) RequestHeader() transport.Header { return tr.headers }

// ReplyHeader 返回测试响应头。
func (tr *testTransport) ReplyHeader() transport.Header { return tr.headers }

type testHeader map[string]string

// Get 读取测试传输头。
func (headers testHeader) Get(key string) string { return headers[key] }

// Set 写入测试传输头。
func (headers testHeader) Set(key, value string) { headers[key] = value }

// Add 追加测试传输头。
func (headers testHeader) Add(key, value string) { headers[key] = value }

// Keys 返回测试传输头名称。
func (headers testHeader) Keys() []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	return keys
}

// Values 返回指定测试传输头的值。
func (headers testHeader) Values(key string) []string {
	if value, ok := headers[key]; ok {
		return []string{value}
	}
	return nil
}

// TestMiddlewareSupportsRateLimitAlgorithms 验证五种算法都能在超限时拒绝请求。
func TestMiddlewareSupportsRateLimitAlgorithms(t *testing.T) {
	store, cleanup, err := cache.NewCache(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	tests := []struct {
		name   string
		policy Policy
	}{
		{name: "token bucket", policy: Policy{Dimension: DimensionGlobal, TokensPerSecond: 0.001, Burst: 1}},
		{name: "fixed window", policy: Policy{Dimension: DimensionGlobal, Algorithm: AlgorithmFixedWindow, Limit: 1, Window: time.Second}},
		{name: "sliding window counter", policy: Policy{Dimension: DimensionGlobal, Algorithm: AlgorithmSlidingWindowCounter, Limit: 1, Window: time.Second}},
		{name: "sliding window log", policy: Policy{Dimension: DimensionGlobal, Algorithm: AlgorithmSlidingWindowLog, Limit: 1, Window: time.Second}},
		{name: "leaky bucket", policy: Policy{Dimension: DimensionGlobal, Algorithm: AlgorithmLeakyBucket, LeakRatePerSecond: 0.001, Capacity: 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation := "/example.v1.ExampleService/" + test.name
			serverTransport := &testTransport{operation: operation, headers: make(testHeader)}
			ctx := transport.NewServerContext(context.Background(), serverTransport)
			called := 0
			handler := NewMiddleware(store, testResolver{policies: []Policy{test.policy}})(func(context.Context, any) (any, error) {
				called++
				return "ok", nil
			})
			if _, err := handler(ctx, nil); err != nil {
				t.Fatalf("first request error = %v", err)
			}
			_, err := handler(ctx, nil)
			if status.Code(err) != codes.ResourceExhausted {
				t.Fatalf("second request code = %s, want %s", status.Code(err), codes.ResourceExhausted)
			}
			if serverTransport.headers["Retry-After"] == "" {
				t.Fatal("limited response is missing Retry-After")
			}
			if called != 1 {
				t.Fatalf("next handler calls = %d, want 1", called)
			}
		})
	}
}

// TestMiddlewareLimitsRequestsByOperation 验证令牌桶突发容量耗尽后返回 RESOURCE_EXHAUSTED。
func TestMiddlewareLimitsRequestsByOperation(t *testing.T) {
	store, cleanup, err := cache.NewCache(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	resolver := testResolver{policies: []Policy{{Dimension: DimensionGlobal, TokensPerSecond: 0.001, Burst: 2}}}
	serverTransport := &testTransport{operation: "/example.v1.ExampleService/Call", headers: make(testHeader)}
	ctx := transport.NewServerContext(context.Background(), serverTransport)
	called := 0
	handler := NewMiddleware(store, resolver)(func(context.Context, any) (any, error) {
		called++
		return "ok", nil
	})
	for range 2 {
		if _, err = handler(ctx, nil); err != nil {
			t.Fatalf("allowed request error = %v", err)
		}
	}
	_, err = handler(ctx, nil)
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("limited request code = %s, want %s", status.Code(err), codes.ResourceExhausted)
	}
	if called != 2 {
		t.Fatalf("next handler calls = %d, want 2", called)
	}
	if serverTransport.headers["Retry-After"] == "" {
		t.Fatal("limited response is missing Retry-After")
	}
}

// TestMiddlewareFailsClosedWhenPolicyCannotLoad 验证策略加载失败时中间件拒绝继续执行。
func TestMiddlewareFailsClosedWhenPolicyCannotLoad(t *testing.T) {
	store, cleanup, err := cache.NewCache(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	serverTransport := &testTransport{operation: "/example.v1.ExampleService/Call", headers: make(testHeader)}
	ctx := transport.NewServerContext(context.Background(), serverTransport)
	handler := NewMiddleware(store, testResolver{err: context.DeadlineExceeded})(func(context.Context, any) (any, error) {
		t.Fatal("handler must not run when policy loading fails")
		return nil, nil
	})
	_, err = handler(ctx, nil)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("policy error code = %s, want %s", status.Code(err), codes.Unavailable)
	}
}

// TestBucketKeyUsesOperationHashTag 验证同一接口的多个限流维度位于 Redis Cluster 同一槽位。
func TestBucketKeyUsesOperationHashTag(t *testing.T) {
	globalKey := bucketKey("/example.v1.ExampleService/Call", DimensionGlobal, "global")
	ipKey := bucketKey("/example.v1.ExampleService/Call", DimensionIP, "192.0.2.1")
	globalTag := strings.SplitN(strings.SplitN(globalKey, "{", 2)[1], "}", 2)[0]
	ipTag := strings.SplitN(strings.SplitN(ipKey, "{", 2)[1], "}", 2)[0]
	if globalTag == "" || globalTag != ipTag {
		t.Fatalf("bucket keys must have the same non-empty hash tag: %q, %q", globalKey, ipKey)
	}
}
