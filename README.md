# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` 是 Kratos 服务的通用运行时。宿主项目负责业务 Case、Service、API 和进程入口；Core 负责基础设施、传输层、资源注册以及应用生命周期。

Core 不是完整的业务模板。宿主通过一个 `module.Module` 把业务服务和构建期资源交给 Core，再由 `NewApp` 统一装配和启动。

## Core 负责什么

- 从 `bootstrap.Context` 解析数据库、Redis、队列、OSS、JWT、翻译器和性能分析配置。
- 按模块资源创建多数据源 GORM 客户端、缓存、队列、OSS、翻译器和共享 `biz.BaseCase`。
- 按配置创建 HTTP、gRPC、MCP、SSE、队列和持久化定时任务运行时，并把模块服务注册到对应传输层。
- 启动时按顺序执行数据库迁移、OpenAPI 接口同步、租户角色菜单同步和 Casbin 策略重建。
- 统一启动、停止 Kratos 服务，并返回应用清理函数。

## 公共边界

跨项目可以依赖的 Go 代码分为四个入口：

- 根包提供 `ProviderSet` 和 `NewApp`。宿主通常只需要把 `ProviderSet` 放入自己的 Wire 图。
- `pkg` 提供 `biz`、`config`、`const`、`dto`、`errorsx` 和 `module` 公共包。
- `api` 是独立 Go 模块，`api/proto` 保存 Core 的 protobuf 定义，`api/gen/go` 保存生成的 Go 类型。
- `client` 是独立 Go 模块，提供基于 `kratos-kit` 配置的 gRPC 连接和进程内 gRPC 连接。

`internal` 下的代码只属于 Core 实现，不是跨项目 API。Core 创建的缓存、队列、OSS、翻译器和多数据源 GORM 客户端会注入 `biz.BaseCase`，同时写入 `kratos-kit/sdk.Runtime`，业务代码可以按需从 BaseCase 或 SDK 获取。

## Wire 接入

宿主在自己的 Wire 组合根中提供一个实现 `module.Module` 的业务模块，并使用 Core 唯一公开的 `ProviderSet`：

```go
//go:build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	core "github.com/liujitcn/kratos-core"
	"github.com/liujitcn/kratos-core/pkg/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

func initializeApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		core.ProviderSet,
		newHostModule,
		wire.Bind(new(module.Module), new(*hostModule)),
	))
}
```

Core 的 `ProviderSet` 只负责把 `NewApp` 接入宿主的 Wire 图；它不会把 `BaseCase`、`Job`、`Docs`、`OpenAPI` 或 `SSE` 作为宿主 Wire 输出。宿主自己的业务 Case 仍应按需引入 `pkg/biz.ProviderSet`、`pkg/config.ProviderSet` 和业务 Provider，但不应直接依赖 Core 的 `internal` 包。`wire_gen.go` 只能通过 `make wire` 或项目自己的 Wire 命令生成，不能手工维护。

## 模块契约

业务模块实现 `pkg/module.Module`。模块自己持有业务 Service，并在协议注册方法中完成注册：

```go
type hostModule struct{}

func (*hostModule) RegisterGRPC(grpc.ServiceRegistrar) {}
func (*hostModule) RegisterHTTP(*kratosHTTP.Server)    {}
func (*hostModule) RegisterMCP(*mcpserver.Server)      {}
func (*hostModule) RegisterQueue(*queueTransport.Server) {}
func (*hostModule) RegisterCron(*cronTransport.Server) error { return nil }
func (*hostModule) RegisterSSE(*sseTransport.Server) error    { return nil }
func (*hostModule) Resources() module.Resources                { return module.Resources{} }
```

各方法的职责如下：

| 方法 | 作用 |
| --- | --- |
| `RegisterGRPC` | 注册生成的 gRPC Service。未配置 gRPC 服务时不会创建 Core gRPC Server。 |
| `RegisterHTTP` | 注册 HTTP Service 和路由。未配置 HTTP 服务时不会创建 Core HTTP Server。 |
| `RegisterMCP` | 注册 MCP 工具。MCP 可以独立监听，也可以挂载到 HTTP。 |
| `RegisterQueue` | 注册队列消费者；Core 同时注册内置日志和任务日志消费者。 |
| `RegisterCron` | 注册数据库持久化任务执行器，通常调用 `server.RegisterTask`。返回错误会中止装配。 |
| `RegisterSSE` | 注册业务 SSE 流，通常调用 `server.RegisterStream`。返回错误会中止装配。 |
| `Resources` | 返回模型、迁移、OpenAPI、项目文档和 I18n 等静态资源。 |

多个业务模块可以作为 `NewApp` 的多个参数传入。Core 会按传入顺序收集资源并转发协议注册；重复文档路径、冲突的 OpenAPI 文档、内容不同的 I18n 消息键或重复 SSE 流标识会在装配时被拒绝。

## 构建期资源

`module.Resources` 是每个模块的一次性资源快照：

| 字段 | 内容与约束 |
| --- | --- |
| `ProjectKey` | 项目稳定标识，用于文档和 OpenAPI 命名；为空时使用 `kratos-core`。 |
| `ProjectName` | 项目展示名称；为空时回退到 `ProjectKey`。 |
| `Models` | 按数据源名称分组的 GORM 模型。含模型的数据源必须在配置中存在，默认数据源必须配置。 |
| `Migrations` | 版本化迁移列表。每项 `module.Migration` 声明 `Name`、`FS`、`Path` 和 `Dependencies`，Core 按依赖顺序执行。 |
| `OpenAPI` | 包含 `openapi.yaml`、`openapi.yml` 或 `openapi.json` 的 `fs.FS`。启用 Swagger 后，Core 会为每个项目挂载原文和 Swagger UI。 |
| `Docs` | 通常包含 `docs.json` 的 `fs.FS`，用于构建项目文档树并通过 `biz.Docs` 查询。 |
| `I18n` | 包含 `zh-CN.json`、`zh-TW.json`、`en-US.json`、`ja-JP.json` 等语言文件的 `fs.FS`。Core 会与内置文案合并。 |

资源通常由宿主通过 `embed.FS`、代码生成器或 `fstest.MapFS` 提供：

```go
func NewModuleResources() module.Resources {
	return module.Resources{
		ProjectKey:  "host",
		ProjectName: "Host Service",
		Models:      map[string][]interface{}{defaultDataSource: models.Models()},
		Docs:        docsFS,
		OpenAPI:     openAPIFS,
		I18n:        i18nFS,
		Migrations: []module.Migration{
			{Name: "host", FS: migrationFS, Path: "."},
		},
	}
}
```

## 运行时能力

### 基础上下文

`pkg/biz.BaseCase` 是宿主业务共用的基础上下文，包含 `bootstrap.Context`、缓存、队列、OSS、翻译器和多数据源 GORM 客户端，并提供 `GetAuthInfo` 读取当前认证用户。

Core 还向宿主暴露以下能力接口：

- `biz.Job`：启动、停止或立即运行数据库中的持久化任务。
- `biz.Docs`：查询合并后的项目文档树和文档正文。
- `biz.OpenAPI`：按服务、HTTP 操作查询 OpenAPI 信息。
- `biz.SSE`：建立 SSE 订阅并发布 JSON 事件。

### 服务与中间件

HTTP 和 gRPC 服务会按配置挂载 request ID、I18n、日志、认证授权和参数校验中间件。HTTP 还支持本地 OSS 静态文件、SPA 回退和 Swagger；启用进程内 MCP 或 SSE 时，对应端点会挂载到 HTTP 服务，因此必须同时配置 HTTP。

队列运行时负责消费 Core 的日志消息和任务日志消息，并转发宿主注册的消费者。Cron 运行时从数据库重载启用的 `BaseJob`，按模块在 `RegisterCron` 中注册的执行器执行任务。

## 启动顺序

`NewApp` 的主要装配顺序如下：

1. 解析启动配置并收集模块资源，根据模块模型创建数据源和迁移注册表。
2. 执行数据库迁移，随后在同一事务中同步 OpenAPI 接口、租户角色菜单和 Casbin 数据库规则；事务提交后刷新内存策略。
3. 创建共享基础服务、认证授权、HTTP/gRPC/MCP/SSE、队列和 Cron 运行时，并调用模块注册方法。
4. 组装 Kratos App；应用运行时统一启动和停止传输服务，返回的清理函数负责释放其余基础资源。

## 目录职责

```text
api/
  proto/common/v1/      Core 公共 protobuf 定义
  gen/go/common/v1/     protobuf 生成的 Go 代码

client/
  connection.go         远程或进程内 gRPC 连接适配
  localgrpc/             进程内 gRPC 服务注册与调用

pkg/
  biz/                   BaseCase 和 Core 能力接口
  config/                启动配置解析
  const/                 公共常量
  dto/                   文档和 OpenAPI 查询 DTO
  errorsx/               统一错误构造
  module/                宿主模块、资源和协议注册契约

internal/
  biz/                   Core 内置业务用例
  data/                  Core 模型、事务和仓储
  job/                   Cron 注册、持久化任务和运行时
  queue/                 队列消费者与生命周期适配
  resource/              文档、I18n、迁移、OpenAPI 和启动资源同步
  server/                HTTP、gRPC、MCP 和中间件
  sse/                   SSE 流注册、传输和发布

bootstrap.go             对外 ProviderSet 和应用生命周期装配
wire.go                  Core 内部 Wire 组合根
wire_gen.go              Wire 生成产物
Makefile                 生成、格式化、测试和静态检查命令
```

## 开发命令

```bash
make api       # 生成 api/gen/go
make wire      # 生成 wire_gen.go
make fmt       # goimports 格式化 Go 代码
make test      # go test ./...
make vet       # go vet ./...
make lint      # 当前等同于 make vet
```

## 发布版本 tag

`scripts/tag_release.py` 参照 `kratos-kit` 按 Go module 独立发布版本。执行前先提交并推送代码，脚本只检查远程默认分支上的提交，不会提交工作区改动：

```bash
git add -A
git commit -m "提交说明"
git push origin main

make tag
```

默认扫描根模块、`api` 和 `client`，只有模块自上一个 tag 后存在已推送的代码更新时才创建并推送下一个 patch tag：

| 模块 | tag 格式 |
| --- | --- |
| 根模块 | `vX.Y.Z` |
| `api` | `api/vX.Y.Z` |
| `client` | `client/vX.Y.Z` |

也可以只处理指定模块：

```bash
MODULE=api make tag              # 从 api 目录开始递归扫描
MODULE=api EXACT=1 make tag      # 只处理 api 模块
```

脚本会自动跳过没有代码更新或 tag 已存在的模块；根模块的变更检测会排除 `api` 和 `client` 子模块，避免子模块改动重复触发根模块 tag。

项目要求 Go `1.26.5`。`api` 和 `client` 是独立 Go 模块，修改它们时还应分别执行 `cd api && go test ./...`、`cd client && go test ./...`。修改公共模块契约后，应额外编译依赖 Core 的宿主项目。

客户端连接的独立说明见 [client/README.md](client/README.md)。
