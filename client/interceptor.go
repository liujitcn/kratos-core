package client

import (
	"fmt"
	"math"
	"time"

	clientMetrics "github.com/liujitcn/kratos-core/client/middleware/metrics"
	clientRateLimit "github.com/liujitcn/kratos-core/client/middleware/ratelimit"
	clientRetry "github.com/liujitcn/kratos-core/client/middleware/retry"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	metricsPrometheus "github.com/liujitcn/kratos-kit/metrics/prometheus"
	"github.com/liujitcn/kratos-kit/ratelimit/tokenbucket"
	coreRetry "github.com/liujitcn/kratos-kit/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/durationpb"
)

const defaultClientMetricsSubsystem = "grpc_client"

type configuredClientInterceptors struct {
	unary  []grpc.UnaryClientInterceptor
	stream []grpc.StreamClientInterceptor
}

// buildConfiguredClientInterceptors 按 metrics、retry、ratelimit 顺序组装配置驱动的客户端拦截器。
func buildConfiguredClientInterceptors(config *configv1.Client_Middleware) (configuredClientInterceptors, error) {
	interceptors := configuredClientInterceptors{}
	if config == nil {
		return interceptors, nil
	}

	var err error
	if config.GetMetrics() != nil {
		err = interceptors.appendMetrics(config.GetMetrics())
		if err != nil {
			return configuredClientInterceptors{}, err
		}
	}
	if config.GetRetry() != nil {
		err = interceptors.appendRetry(config.GetRetry())
		if err != nil {
			return configuredClientInterceptors{}, err
		}
	}
	if config.GetRateLimiter() != nil {
		err = interceptors.appendRateLimiter(config.GetRateLimiter())
		if err != nil {
			return configuredClientInterceptors{}, err
		}
	}
	return interceptors, nil
}

// appendMetrics 创建 Prometheus 指标提供者并追加 unary 与 stream 指标拦截器。
func (interceptors *configuredClientInterceptors) appendMetrics(config *configv1.Client_Middleware_Metrics) error {
	providerOptions := make([]metricsPrometheus.Option, 0, 2)
	if config.GetNamespace() != "" {
		providerOptions = append(providerOptions, metricsPrometheus.WithNamespace(config.GetNamespace()))
	}
	subsystem := config.GetSubsystem()
	if subsystem == "" {
		subsystem = defaultClientMetricsSubsystem
	}
	providerOptions = append(providerOptions, metricsPrometheus.WithSubsystem(subsystem))

	provider, err := metricsPrometheus.NewWithDefaultRegistry(providerOptions...)
	if err != nil {
		return fmt.Errorf("创建客户端 Prometheus 指标提供者: %w", err)
	}
	options := make([]clientMetrics.Option, 0, 4)
	if config.GetRequestCounterName() != "" {
		options = append(options, clientMetrics.WithRequestCounterName(config.GetRequestCounterName()))
	}
	if config.GetLatencyHistogramName() != "" {
		options = append(options, clientMetrics.WithLatencyHistogramName(config.GetLatencyHistogramName()))
	}
	if config.GetInFlightGaugeName() != "" {
		options = append(options, clientMetrics.WithInFlightGaugeName(config.GetInFlightGaugeName()))
	}
	if len(config.GetSkipMethods()) > 0 {
		skippedMethods := make(map[string]struct{}, len(config.GetSkipMethods()))
		for _, method := range config.GetSkipMethods() {
			skippedMethods[method] = struct{}{}
		}
		options = append(options, clientMetrics.WithSkipFunc(func(method string) bool {
			_, skipped := skippedMethods[method]
			return skipped
		}))
	}
	interceptors.unary = append(interceptors.unary, clientMetrics.UnaryClientInterceptor(provider, options...))
	interceptors.stream = append(interceptors.stream, clientMetrics.StreamClientInterceptor(provider, options...))
	return nil
}

// appendRetry 根据配置创建退避策略并追加一元 RPC 重试拦截器。
func (interceptors *configuredClientInterceptors) appendRetry(config *configv1.Client_Middleware_Retry) error {
	initialBackoff := 200 * time.Millisecond
	maxBackoff := 10 * time.Second
	backoffFactor := 2.0
	var err error
	if config.GetInitialBackoff() != nil {
		initialBackoff, err = positiveDuration(config.GetInitialBackoff(), "retry.initial_backoff")
		if err != nil {
			return err
		}
	}
	if config.GetMaxBackoff() != nil {
		maxBackoff, err = positiveDuration(config.GetMaxBackoff(), "retry.max_backoff")
		if err != nil {
			return err
		}
	}
	if math.IsNaN(config.GetBackoffFactor()) || math.IsInf(config.GetBackoffFactor(), 0) || config.GetBackoffFactor() < 0 {
		return fmt.Errorf("客户端配置 retry.backoff_factor 必须为有限非负数")
	}
	if config.GetBackoffFactor() > 0 {
		backoffFactor = config.GetBackoffFactor()
	}
	retrierOptions := []coreRetry.Option{coreRetry.WithBackoff(coreRetry.ExponentialBackoff{
		Initial: initialBackoff,
		Factor:  backoffFactor,
		Max:     maxBackoff,
	})}
	if config.GetMaxAttempts() > 0 {
		retrierOptions = append(retrierOptions, coreRetry.WithMaxAttempts(int(config.GetMaxAttempts())))
	}
	if config.GetMaxTotalWait() != nil {
		var maxTotalWait time.Duration
		maxTotalWait, err = positiveDuration(config.GetMaxTotalWait(), "retry.max_total_wait")
		if err != nil {
			return err
		}
		retrierOptions = append(retrierOptions, coreRetry.WithMaxTotalWait(maxTotalWait))
	}

	interceptorOptions := make([]clientRetry.Option, 0, 3)
	if len(config.GetIdempotentPrefixes()) > 0 {
		interceptorOptions = append(interceptorOptions, clientRetry.WithIdempotentPrefixes(config.GetIdempotentPrefixes()...))
	}
	if len(config.GetRetryCodes()) > 0 {
		var retryCodes []codes.Code
		retryCodes, err = parseRetryCodes(config.GetRetryCodes())
		if err != nil {
			return err
		}
		interceptorOptions = append(interceptorOptions, clientRetry.WithRetryCodes(retryCodes...))
	}
	if len(config.GetSkipMethods()) > 0 {
		interceptorOptions = append(interceptorOptions, clientRetry.WithSkipMethods(config.GetSkipMethods()...))
	}
	interceptors.unary = append(interceptors.unary, clientRetry.UnaryClientInterceptor(coreRetry.New(retrierOptions...), interceptorOptions...))
	return nil
}

// appendRateLimiter 创建令牌桶并追加 unary 与 stream 限流拦截器。
func (interceptors *configuredClientInterceptors) appendRateLimiter(config *configv1.Client_Middleware_RateLimiter) error {
	if math.IsNaN(config.GetTokensPerSecond()) || math.IsInf(config.GetTokensPerSecond(), 0) || config.GetTokensPerSecond() <= 0 {
		return fmt.Errorf("客户端配置 rate_limiter.tokens_per_second 必须为有限正数")
	}
	if config.GetBurst() == 0 {
		return fmt.Errorf("客户端配置 rate_limiter.burst 必须大于 0")
	}
	limiter, err := tokenbucket.New(config.GetTokensPerSecond(), int(config.GetBurst()))
	if err != nil {
		return fmt.Errorf("创建客户端令牌桶限流器: %w", err)
	}
	options := make([]clientRateLimit.Option, 0, 2)
	if config.GetWait() {
		options = append(options, clientRateLimit.WithWait())
	}
	if len(config.GetSkipMethods()) > 0 {
		options = append(options, clientRateLimit.WithSkipMethods(config.GetSkipMethods()...))
	}
	interceptors.unary = append(interceptors.unary, clientRateLimit.UnaryClientInterceptor(limiter, options...))
	interceptors.stream = append(interceptors.stream, clientRateLimit.StreamClientInterceptor(limiter, options...))
	return nil
}

// positiveDuration 校验 protobuf duration 并返回严格大于零的时长。
func positiveDuration(value *durationpb.Duration, field string) (time.Duration, error) {
	err := value.CheckValid()
	if err != nil {
		return 0, fmt.Errorf("客户端配置 %s 无效: %w", field, err)
	}
	duration := value.AsDuration()
	if duration <= 0 {
		return 0, fmt.Errorf("客户端配置 %s 必须大于 0", field)
	}
	return duration, nil
}

// parseRetryCodes 将配置中的 gRPC 状态码名称转换为 codes.Code。
func parseRetryCodes(names []string) ([]codes.Code, error) {
	codesByName := map[string]codes.Code{
		"CANCELED":            codes.Canceled,
		"UNKNOWN":             codes.Unknown,
		"INVALID_ARGUMENT":    codes.InvalidArgument,
		"DEADLINE_EXCEEDED":   codes.DeadlineExceeded,
		"NOT_FOUND":           codes.NotFound,
		"ALREADY_EXISTS":      codes.AlreadyExists,
		"PERMISSION_DENIED":   codes.PermissionDenied,
		"RESOURCE_EXHAUSTED":  codes.ResourceExhausted,
		"FAILED_PRECONDITION": codes.FailedPrecondition,
		"ABORTED":             codes.Aborted,
		"OUT_OF_RANGE":        codes.OutOfRange,
		"UNIMPLEMENTED":       codes.Unimplemented,
		"INTERNAL":            codes.Internal,
		"UNAVAILABLE":         codes.Unavailable,
		"DATA_LOSS":           codes.DataLoss,
		"UNAUTHENTICATED":     codes.Unauthenticated,
	}
	retryCodes := make([]codes.Code, 0, len(names))
	for _, name := range names {
		code, ok := codesByName[name]
		if !ok {
			return nil, fmt.Errorf("客户端配置 retry.retry_codes 包含未知状态码 %q", name)
		}
		retryCodes = append(retryCodes, code)
	}
	return retryCodes, nil
}
