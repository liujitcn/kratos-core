package requestid

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestUnaryClientInterceptorInjectsRequestID 验证一元客户端拦截器会注入请求标识。
func TestUnaryClientInterceptorInjectsRequestID(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithRequestIDGenerator(func() string { return "generated" }))
	var got string
	err := interceptor(context.Background(), "/test.Service/Call", nil, nil, nil, func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		outgoing, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("outgoing metadata is missing")
		}
		got = outgoing.Get(DefaultRequestIDHeader)[0]
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if got != "generated" {
		t.Fatalf("request id = %q, want generated", got)
	}
}

// TestStreamClientInterceptorInjectsRequestID 验证流式客户端拦截器会在建流前注入请求标识。
func TestStreamClientInterceptorInjectsRequestID(t *testing.T) {
	interceptor := StreamClientInterceptor(WithRequestIDGenerator(func() string { return "stream" }))
	var got string
	_, err := interceptor(context.Background(), &grpc.StreamDesc{}, nil, "/test.Service/Watch", func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
		outgoing, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("outgoing metadata is missing")
		}
		got = outgoing.Get(DefaultRequestIDHeader)[0]
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if got != "stream" {
		t.Fatalf("request id = %q, want stream", got)
	}
}
