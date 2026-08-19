# Admin Client

`client` 只负责根据 `kratos-kit/api` 的客户端配置初始化 gRPC 连接，不封装具体业务客户端。

```go
connection, cleanup, err := client.NewConnection(ctx, clientConfig)
if err != nil {
	return err
}
defer cleanup()

userClient := systemadminv1.NewBaseUserServiceClient(connection)
response, err := userClient.GetBaseUser(ctx, request)
```

配置示例：

```yaml
client:
  grpc:
    # 本地或远程直连地址。
    endpoint: 127.0.0.1:6001
    middleware:
      retry:
        max_attempts: 3
        initial_backoff: 200ms
        max_backoff: 10s
        backoff_factor: 2
        idempotent_prefixes: [Get, List, Search]
        retry_codes: [UNAVAILABLE]
      rate_limiter:
        tokens_per_second: 100
        burst: 200
        wait: false
      metrics:
        namespace: application
        subsystem: grpc_client
```

普通地址用于本机或远程直连。使用 `discovery:///服务名` 时，通过 `client.WithDiscovery(discovery)` 注入注册中心发现器。`timeout`、`tls`、静态 `metadata`、JWT、熔断、日志、链路追踪和负载均衡等配置由当前 `client` 模块统一处理；额外的 Kratos 客户端中间件通过 `client.WithMiddleware` 传入，原生 gRPC 客户端拦截器通过 `client.WithUnaryInterceptor` 或 `client.WithStreamInterceptor` 传入。客户端中间件按请求 ID、恢复、追踪、metadata、日志、熔断、认证、业务自定义的顺序组装；配置拦截器按 metrics、retry、ratelimit 顺序执行，使指标覆盖完整逻辑调用，并让每次重试尝试都经过限流。流式调用不自动重试，只启用 metrics 和 ratelimit，并在建流前执行一次 middleware、传播 transport 请求头。负载均衡器是进程级配置，同一进程中的连接必须使用同一种策略。

`retry`、`rate_limiter`、`metrics` 配置对象存在即启用对应拦截器。retry 未配置具体参数时使用 3 次尝试、200ms 初始退避、2 倍退避和 10s 单次退避上限，只重试 `Get`、`List`、`Search` 前缀方法返回的 `UNAVAILABLE`。rate_limiter 使用令牌桶，启用时 `tokens_per_second` 和 `burst` 必须大于 0。metrics 注册到进程默认 Prometheus Registry，`subsystem` 为空时使用 `grpc_client`。

```go
connection, cleanup, err := client.NewConnection(
	ctx,
	clientConfig,
	client.WithMiddleware(customMiddleware),
	client.WithUnaryInterceptor(customUnaryInterceptor),
	client.WithStreamInterceptor(customStreamInterceptor),
)
if err != nil {
	return err
}
defer cleanup()
```

当 `client`、`client.grpc` 或 `client.grpc.endpoint` 为空时，`NewConnection` 创建进程内客户端，不建立网络连接。需要调用进程内服务时，通过 `WithLocalServices` 注册服务：

```go
connection, cleanup, err := client.NewConnection(ctx, nil, client.WithLocalServices(modules.RegisterGRPC))
if err != nil {
	return err
}
defer cleanup()
```
