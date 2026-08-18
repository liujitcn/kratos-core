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
```

普通地址用于本机或远程直连。使用 `discovery:///服务名` 时，通过 `client.WithDiscovery(discovery)` 注入注册中心发现器。`timeout`、`tls`、`metadata`、JWT、熔断、链路追踪和负载均衡等配置由当前 `client` 模块统一处理；额外的 Kratos 客户端中间件通过 `client.WithMiddleware` 传入。负载均衡器是进程级配置，同一进程中的连接必须使用同一种策略。

```go
connection, cleanup, err := client.NewConnection(
	ctx,
	clientConfig,
	client.WithMiddleware(customMiddleware),
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
