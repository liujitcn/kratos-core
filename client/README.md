# gRPC Client

`client` 是 `kratos-core` 的独立 gRPC 客户端模块。它统一处理远程直连、服务发现和进程内调用，并让三种连接模式共享请求标识、metadata、认证、日志、追踪、熔断、重试、限流和指标能力。模块不生成具体业务客户端，业务代码仍使用 protobuf 生成的客户端接口。

## 创建连接

```go
connection, cleanup, err := client.NewConnection(ctx, clientConfig)
if err != nil {
	return err
}
defer cleanup()

response, err := systemadminv1.NewBaseUserServiceClient(connection).GetBaseUser(ctx, request)
```

`NewConnection` 返回实现 `grpc.ClientConnInterface` 的 `Connection` 和清理函数。连接初始化失败会返回错误；远程连接的清理函数负责关闭底层连接，进程内连接的清理函数为空操作。

如果调用方已经持有完整启动配置，也可以使用 `CreateGrpcClient` 直接创建底层 gRPC 连接，并按服务名或配置地址解析 endpoint；`NewConnection` 适合同时处理远程和进程内两种模式。

## 连接模式

### 远程直连

当客户端配置包含 gRPC endpoint 时，普通地址按直连方式使用，例如 `127.0.0.1:6001`。带有协议前缀的地址会原样交给底层传输层处理。未配置 endpoint 时，`NewConnection` 不会建立网络连接，而是创建进程内连接。

### 服务发现

服务名会转换为 `discovery:///服务名`。使用该地址时必须通过 `client.WithDiscovery` 注入服务发现器，否则连接创建失败。服务发现和负载均衡器属于进程级配置；同一进程中的连接必须使用同一种负载均衡策略。

```go
connection, cleanup, err := client.NewConnection(
	ctx,
	clientConfig,
	client.WithDiscovery(discovery),
)
```

支持的负载均衡策略为 `wrr`、`p2c` 和 `random`，默认使用 `wrr`；通过客户端配置选择后，运行期间不能切换。

### 进程内调用

当 `client`、`client.grpc` 或 `client.grpc.endpoint` 为空时，使用 `client.WithLocalServices` 注册生成代码提供的服务：

```go
connection, cleanup, err := client.NewConnection(
	ctx,
	nil,
	client.WithLocalServices(modules.RegisterGRPC),
)
if err != nil {
	return err
}
defer cleanup()
```

进程内连接同时实现 unary 和 streaming gRPC 调用，执行客户端与服务端拦截器，并传播请求头、响应 header 和 trailer。调用未注册的方法返回 `Unimplemented`，服务处理器 panic 会转换为错误。

## 配置与拦截器

配置对象存在时启用对应能力。常用配置示例：

```yaml
client:
  grpc:
    endpoint: 127.0.0.1:6001
    timeout: 5s
    middleware:
      enable_logging: true
      enable_tracing: true
      enable_metadata: true
      enable_circuit_breaker: true
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

客户端中间件按请求标识、恢复、追踪、metadata、日志、熔断、认证和自定义顺序组装。配置型 gRPC 拦截器按 metrics、retry、ratelimit 参与 unary 调用；stream 调用只在建流时执行 middleware 和限流，并记录完整流生命周期的指标，不自动重试。

- **超时与 TLS**：未配置 timeout 时默认 5 秒；配置 TLS 时加载客户端证书和校验设置。
- **静态 metadata**：写入每次请求的 transport header；stream 建立时会一并传播。
- **认证**：配置 JWT 后为请求生成认证 metadata；未提供密钥时从已初始化的运行时密钥派生。
- **请求标识**：默认使用 `X-Request-Id`，已有标识会透传，没有时生成新的标识。
- **日志与指标**：记录方法、目标、状态码和耗时；指标包含请求总数、耗时直方图和进行中请求数，stream 在 EOF 或错误时完成记录。

### 重试

重试只适用于 unary 调用，并且只对幂等方法执行。默认识别方法名 `Get`、`List`、`Search` 前缀，默认仅重试 `UNAVAILABLE`，最多 3 次尝试，初始退避 200ms、倍数 2、单次退避上限 10 秒。可以通过配置或 `client/middleware/retry` 的选项修改方法前缀、状态码和跳过列表。

### 限流

令牌桶限流支持立即失败和等待两种模式。无法取得令牌时返回 `ResourceExhausted`；等待模式会在上下文取消时结束。可以通过 `client/middleware/ratelimit` 的选项跳过指定方法。

## 自定义扩展

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

`WithMiddleware` 追加 Kratos 客户端 middleware；`WithUnaryInterceptor` 和 `WithStreamInterceptor` 追加原生 gRPC 客户端拦截器。自定义拦截器在配置型拦截器之后执行。

## 包目录

```text
client/connection.go             Connection、连接模式和 Option
client/grpc.go                   endpoint、TLS、服务发现和负载均衡
client/middleware.go             请求标识、认证、日志和通用 middleware
client/interceptor.go            配置型 unary/stream 拦截器组装
client/middleware/requestid/     请求标识 middleware 和拦截器
client/middleware/retry/         unary 重试拦截器
client/middleware/ratelimit/     unary/stream 限流拦截器
client/middleware/metrics/       unary/stream 指标拦截器
client/localgrpc/                进程内 gRPC 服务注册、调用和元数据
```

修改 `client` 模块后，在仓库根目录执行 `make test` 或单独执行：

```bash
cd client
go test ./...
go vet ./...
```
