package metrics

import (
	"context"
	"io"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testRecorder struct {
	counters  map[string]int64
	gauges    map[string]float64
	lastCode  string
	histogram int
}

// Counter 记录测试计数器与状态码标签。
func (recorder *testRecorder) Counter(_ context.Context, name string, value int64, labels map[string]string) {
	if recorder.counters == nil {
		recorder.counters = make(map[string]int64)
	}
	recorder.counters[name] += value
	recorder.lastCode = labels["code"]
}

// Histogram 记录测试直方图调用次数。
func (recorder *testRecorder) Histogram(context.Context, string, float64, map[string]string) {
	recorder.histogram++
}

// Gauge 记录测试仪表盘最终值。
func (recorder *testRecorder) Gauge(_ context.Context, name string, value float64, _ map[string]string) {
	if recorder.gauges == nil {
		recorder.gauges = make(map[string]float64)
	}
	recorder.gauges[name] = value
}

// TestUnaryClientInterceptorRecordsResult 验证一元请求会记录状态、耗时和进行中指标。
func TestUnaryClientInterceptorRecordsResult(t *testing.T) {
	recorder := new(testRecorder)
	interceptor := UnaryClientInterceptor(recorder)
	invoker := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		return status.Error(codes.NotFound, "missing")
	}
	err := interceptor(context.Background(), "/test.Service/GetValue", nil, nil, nil, invoker)
	if status.Code(err) != codes.NotFound {
		t.Fatalf("error code = %s, want %s", status.Code(err), codes.NotFound)
	}
	if recorder.counters[defaultRequestCounter] != 1 || recorder.histogram != 1 {
		t.Fatalf("counter = %d, histogram calls = %d", recorder.counters[defaultRequestCounter], recorder.histogram)
	}
	if recorder.gauges[defaultInFlightGauge] != 0 {
		t.Fatalf("in-flight gauge = %f, want 0", recorder.gauges[defaultInFlightGauge])
	}
	if recorder.lastCode != codes.NotFound.String() {
		t.Fatalf("code label = %q, want %q", recorder.lastCode, codes.NotFound.String())
	}
}

// TestStreamClientInterceptorRecordsCompletion 验证流读取结束时会完成指标记录。
func TestStreamClientInterceptorRecordsCompletion(t *testing.T) {
	recorder := new(testRecorder)
	interceptor := StreamClientInterceptor(recorder)
	streamer := func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
		return &completedClientStream{}, nil
	}
	stream, err := interceptor(context.Background(), new(grpc.StreamDesc), nil, "/test.Service/Watch", streamer)
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	err = stream.RecvMsg(new(struct{}))
	if err != io.EOF {
		t.Fatalf("RecvMsg() error = %v, want EOF", err)
	}
	if recorder.counters[defaultRequestCounter] != 1 || recorder.histogram != 1 {
		t.Fatalf("counter = %d, histogram calls = %d", recorder.counters[defaultRequestCounter], recorder.histogram)
	}
	if recorder.gauges[defaultInFlightGauge] != 0 {
		t.Fatalf("in-flight gauge = %f, want 0", recorder.gauges[defaultInFlightGauge])
	}
}

type completedClientStream struct {
	grpc.ClientStream
}

// RecvMsg 返回 EOF，模拟服务端正常结束流。
func (*completedClientStream) RecvMsg(any) error {
	return io.EOF
}
