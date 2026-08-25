// Package metrics 提供 gRPC 客户端请求指标拦截器。
package metrics

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/liujitcn/kratos-kit/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	defaultRequestCounter   = "grpc_requests_total"
	defaultLatencyHistogram = "grpc_request_duration_seconds"
	defaultInFlightGauge    = "grpc_requests_in_flight"
)

// Option 配置客户端指标拦截器。
type Option func(*options)

type options struct {
	requestCounter   string
	latencyHistogram string
	inFlightGauge    string
	skipFunc         func(string) bool
}

// WithRequestCounterName 设置请求计数器指标名称。
func WithRequestCounterName(name string) Option {
	return func(options *options) {
		options.requestCounter = name
	}
}

// WithLatencyHistogramName 设置耗时直方图指标名称。
func WithLatencyHistogramName(name string) Option {
	return func(options *options) {
		options.latencyHistogram = name
	}
}

// WithInFlightGaugeName 设置进行中请求指标名称。
func WithInFlightGaugeName(name string) Option {
	return func(options *options) {
		options.inFlightGauge = name
	}
}

// WithSkipFunc 设置需要跳过指标记录的方法判断函数。
func WithSkipFunc(skip func(string) bool) Option {
	return func(options *options) {
		options.skipFunc = skip
	}
}

// UnaryClientInterceptor 创建记录出站一元 RPC 指标的客户端拦截器。
func UnaryClientInterceptor(recorder metrics.Metrics, interceptorOptions ...Option) grpc.UnaryClientInterceptor {
	config := newOptions(interceptorOptions)
	return func(ctx context.Context, method string, request any, reply any, conn *grpc.ClientConn, invoker grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		if config.skipFunc != nil && config.skipFunc(method) {
			return invoker(ctx, method, request, reply, conn, callOptions...)
		}
		gaugeLabels := labels(method, nil)
		recorder.Gauge(ctx, config.inFlightGauge, 1, gaugeLabels)
		defer recorder.Gauge(ctx, config.inFlightGauge, 0, gaugeLabels)

		start := time.Now()
		err := invoker(ctx, method, request, reply, conn, callOptions...)
		finish(recorder, config, ctx, method, start, err)
		return err
	}
}

// StreamClientInterceptor 创建记录出站流式 RPC 指标的客户端拦截器。
func StreamClientInterceptor(recorder metrics.Metrics, interceptorOptions ...Option) grpc.StreamClientInterceptor {
	config := newOptions(interceptorOptions)
	return func(ctx context.Context, description *grpc.StreamDesc, conn *grpc.ClientConn, method string, streamer grpc.Streamer, callOptions ...grpc.CallOption) (grpc.ClientStream, error) {
		if config.skipFunc != nil && config.skipFunc(method) {
			return streamer(ctx, description, conn, method, callOptions...)
		}
		gaugeLabels := labels(method, nil)
		recorder.Gauge(ctx, config.inFlightGauge, 1, gaugeLabels)
		start := time.Now()
		stream, err := streamer(ctx, description, conn, method, callOptions...)
		if err != nil {
			finishStream(recorder, config, ctx, method, start, gaugeLabels, err)
			return nil, err
		}
		return &trackedClientStream{
			ClientStream: stream,
			recorder:     recorder,
			config:       config,
			ctx:          ctx,
			method:       method,
			start:        start,
			gaugeLabels:  gaugeLabels,
		}, nil
	}
}

type trackedClientStream struct {
	grpc.ClientStream
	recorder    metrics.Metrics
	config      *options
	ctx         context.Context
	method      string
	start       time.Time
	gaugeLabels map[string]string
	once        sync.Once
}

// Header 获取服务端响应头，并在失败时完成流式指标记录。
func (stream *trackedClientStream) Header() (metadata.MD, error) {
	header, err := stream.ClientStream.Header()
	if err != nil {
		stream.finish(err)
	}
	return header, err
}

// RecvMsg 接收流消息，并在流结束时完成指标记录。
func (stream *trackedClientStream) RecvMsg(message any) error {
	err := stream.ClientStream.RecvMsg(message)
	if err == io.EOF {
		stream.finish(nil)
	} else if err != nil {
		stream.finish(err)
	}
	return err
}

// SendMsg 发送流消息，并在失败时完成指标记录。
func (stream *trackedClientStream) SendMsg(message any) error {
	err := stream.ClientStream.SendMsg(message)
	if err != nil {
		stream.finish(err)
	}
	return err
}

// finish 以幂等方式记录流的最终指标。
func (stream *trackedClientStream) finish(err error) {
	stream.once.Do(func() {
		finishStream(stream.recorder, stream.config, stream.ctx, stream.method, stream.start, stream.gaugeLabels, err)
	})
}

// newOptions 创建带默认指标名称的拦截器配置。
func newOptions(interceptorOptions []Option) *options {
	config := &options{
		requestCounter:   defaultRequestCounter,
		latencyHistogram: defaultLatencyHistogram,
		inFlightGauge:    defaultInFlightGauge,
	}
	for _, option := range interceptorOptions {
		option(config)
	}
	return config
}

// finish 记录一次一元 RPC 的请求数和耗时。
func finish(recorder metrics.Metrics, config *options, ctx context.Context, method string, start time.Time, err error) {
	metricLabels := labels(method, err)
	recorder.Counter(ctx, config.requestCounter, 1, metricLabels)
	recorder.Histogram(ctx, config.latencyHistogram, time.Since(start).Seconds(), metricLabels)
}

// finishStream 记录流式 RPC 的最终结果并清除进行中指标。
func finishStream(recorder metrics.Metrics, config *options, ctx context.Context, method string, start time.Time, gaugeLabels map[string]string, err error) {
	finish(recorder, config, ctx, method, start, err)
	recorder.Gauge(ctx, config.inFlightGauge, 0, gaugeLabels)
}

// labels 返回标准 gRPC 客户端指标标签。
func labels(method string, err error) map[string]string {
	code := codes.OK.String()
	if err != nil {
		code = status.Code(err).String()
	}
	return map[string]string{
		"method":  method,
		"service": serviceFromMethod(method),
		"code":    code,
	}
}

// serviceFromMethod 从完整 gRPC 方法名提取服务名。
func serviceFromMethod(fullMethod string) string {
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return fullMethod
}
