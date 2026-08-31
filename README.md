# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` 是 Kratos 服务的通用运行时。宿主项目负责业务 Case、Service、API 和进程入口；Core 负责基础设施、传输层、资源注册以及应用生命周期。

Core 不是完整的业务模板。宿主通过一个 `module.Module` 把业务服务和构建期资源交给 Core，再由 `NewApp` 统一装配并交给宿主启动。

## Core 负责什么

- 从 `bootstrap.Context` 解析数据库、Redis、队列、OSS、JWT、翻译器和性能分析配置。
- 按模块资源创建多数据源 GORM 客户端、缓存、队列、OSS、翻译器和共享 `biz.BaseCase`。
- 按配置创建 HTTP、gRPC、MCP、SSE、队列和持久化定时任务运行时，并把模块服务注册到对应传输层；任务、SSE 流和队列消费者分别作为独立集合注入。
- 装配时按顺序执行数据库迁移、OpenAPI 接口同步、租户角色菜单同步和 Casbin 策略重建。
- 统一装配可选服务并交给 Kratos 管理生命周期；基础资源由 Wire 生成的清理函数释放。

## 公共边界

跨项目可以依赖的 Go 代码分为四类入口：

- 根包提供 `ProviderSet` 和 `NewApp`。宿主通常只需要把 `ProviderSet` 放入自己的 Wire 图。
- 根目录的 `audit`、`biz`、`config`、`const`、`data`、`errorsx`、`job`、`mcp`、`module`、`queue`、`resource`、`server` 和 `sse` 提供跨项目公共包。
- `api` 是独立 Go 模块，`api/proto` 保存 Core 的 protobuf 定义，`api/gen/go` 保存生成的 Go 类型。
- `client` 是独立 Go 模块，提供基于 `kratos-kit` 配置的 gRPC 连接和进程内 gRPC 连接。

`internal/models` 下的代码只属于 Core 实现，不是跨项目 API。Core 创建的缓存、队列、OSS、翻译器和多数据源 GORM 客户端会注入 `biz.BaseCase`，同时写入 `kratos-kit/sdk.Runtime`，业务代码可以按需从 BaseCase 或 SDK 获取。

## Wire 接入

宿主在自己的 Wire 组合根中提供一个实现 `module.Module` 的业务模块，并使用 Core 唯一公开的 `ProviderSet`：

```go
//go:build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	core "github.com/liujitcn/kratos-core"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

func initializeApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		core.ProviderSet,
		newHostModule,
		newHostModules,
	))
}

func newHostModules(host *hostModule) []module.Module {
	return []module.Module{host}
}
```

Core 的 `ProviderSet` 汇总配置、基础设施、模块资源、数据访问、资源同步和各协议运行时，并包含 `NewApp`。宿主只需补充自己的业务 Provider，并提供 `[]module.Module`，不需要重复加入 Core 的 ProviderSet。Core 根目录不再维护 `wire.go` 或 `wire_gen.go`；宿主项目应通过自己的 Wire 命令生成组合根和 `wire_gen.go`，也可以使用 `make wire WIRE_DIR=<宿主 Wire 目录>`。

## 模块契约

业务模块实现 `module.Module`。模块自己持有业务 Service，并在协议注册方法中完成 HTTP、gRPC 和 MCP 注册：

```go
type hostModule struct{}

func (*hostModule) RegisterGRPC(grpc.ServiceRegistrar) {}
func (*hostModule) RegisterHTTP(*kratosHTTP.Server)    {}
func (*hostModule) RegisterMCP(*mcpserver.Server)      {}
```

各方法的职责如下：

| 方法 | 作用 |
| --- | --- |
| `RegisterGRPC` | 注册生成的 gRPC Service。未配置 gRPC 服务时不会创建 Core gRPC Server。 |
| `RegisterHTTP` | 注册 HTTP Service 和路由。未配置 HTTP 服务时不会创建 Core HTTP Server。 |
| `RegisterMCP` | 注册 MCP 工具。MCP 可以独立监听，也可以挂载到 HTTP。 |

多个业务模块可以由宿主 Wire 组合根作为 `module.Module` 提供。任务通过 `job.Tasks` 提供，SSE 流通过 `sse.Streams` 提供，队列消费者通过 `queue.Consumers` 提供；没有对应能力时提供空集合。Core 会按提供顺序转发协议注册，并在装配时校验重复资源和重复 SSE 流标识。

## 构建期资源

`module.Resource` 是宿主提供的一组资源接口，`module.Resources` 聚合多个资源实现：

| 方法 | 内容与约束 |
| --- | --- |
| `ProjectKey()` / `ProjectName()` | 项目稳定标识和展示名称；ProjectKey 为空时使用 `kratos-core`，ProjectName 为空时回退到 ProjectKey。 |
| `Models()` | 按数据源名称分组的 GORM 模型。含模型的数据源必须在配置中存在，默认数据源必须配置。 |
| `Migrations()` | 版本化迁移列表。每项 `module.Migration` 声明 `Name`、`FS`、`Path` 和 `Dependencies`，Core 按依赖顺序执行；每个数据库类型或命名数据源目录使用 `README.md` 作为主说明，可选的 `README.<locale>.md` 由宿主同步。 |
| `OpenAPI()` / `Docs()` / `I18n()` | 分别返回 OpenAPI、项目文档和语言 JSON 文件系统；项目文档包含 `docs.json` 和可选的 `docs.<locale>.json`，未提供的资源返回 nil。 |

资源通常由宿主通过 `embed.FS`、代码生成器或 `fstest.MapFS` 提供：

```go
type hostResources struct{}

func (*hostResources) ProjectKey() string                    { return "host" }
func (*hostResources) ProjectName() string                   { return "Host Service" }
func (*hostResources) Models() module.Models                 { return map[string][]interface{}{defaultDataSource: models.Models()} }
func (*hostResources) Docs() fs.FS                            { return docsFS }
func (*hostResources) OpenAPI() fs.FS                        { return openAPIFS }
func (*hostResources) I18n() fs.FS                           { return i18nFS }
func (*hostResources) Migrations() module.Migrations          { return module.Migrations{{Name: "host", FS: migrationFS, Path: "."}} }

func NewModuleResources() module.Resources { return module.Resources{&hostResources{}} }
```

迁移说明目录结构：

```text
assets/<version>/<database-type>/README.md
assets/<version>/<database-type>/README.<locale>.md
assets/<version>/<database-type>/<data-source>/README.md
assets/<version>/<database-type>/<data-source>/README.<locale>.md
```

Core 只把 `README.md` 交给迁移执行器写入 `base_migration.description`；迁移执行完成后，将现有 `README.<locale>.md` 按 `target_type=7`、迁移记录 ID 写入 `base_i18n`。宿主需要通过模型自动迁移提供该表；缺少某个 `README.<locale>.md` 时不会生成该语言记录。

## 运行时能力

### 基础上下文

`biz.BaseCase` 是宿主业务共用的基础上下文，包含 `bootstrap.Context`、缓存、队列、OSS、翻译器和多数据源 GORM 客户端，并提供 `GetAuthInfo` 读取当前认证用户。

Core 还向宿主提供以下具体业务服务：

- `job.Job`：启动、停止或立即运行数据库中的持久化任务。
- `resource/docs.Docs`：按请求语言选择完整的项目文档目录树、文档显示名和正文；稳定路径不随语言变化，语言文件缺失时按基础语言、默认 `docs.json` 依次回退。
- `resource/openapi.OpenAPI`：按请求语言、服务或 HTTP 操作查询 OpenAPI 信息。
- `sse.SSE`：建立 SSE 订阅并发布 JSON 事件。

### 服务与中间件

HTTP 和 gRPC 服务会按配置挂载 request ID、I18n、日志、认证授权和参数校验中间件。HTTP 还支持本地 OSS 静态文件、SPA 回退和 Swagger；启用进程内 MCP 或 SSE 时，对应端点会挂载到 HTTP 服务，因此必须同时配置 HTTP。

队列运行时负责消费任务日志消息并转发宿主通过 `queue.Consumers` 提供的业务事件消费者；Core 审计流水线负责投递并异步写入 API 访问和策略评估日志，完整模型注册与自动迁移仍由宿主提供。Cron 运行时从数据库重载启用的 `BaseJob`，按 `job.Tasks` 中的执行器执行任务。

## 装配与启动顺序

`ProviderSet` 与 `NewApp` 的主要装配顺序如下：

1. 解析启动配置并收集模块资源，根据模块模型创建数据源和迁移注册表。
2. 执行数据库迁移，随后在同一事务中同步 OpenAPI 接口、`base_api_i18n` 语言快照、租户角色菜单和 Casbin 数据库规则；接口语言记录使用 `operation + locale` 唯一键，不关联会变化的 `base_api.id`；事务提交后刷新内存策略。
3. 创建共享基础服务、认证授权、HTTP/gRPC/MCP/SSE、队列和 Cron 运行时，并注册独立的任务、SSE 流和队列消费者集合。
4. 组装 Kratos App；Kratos 统一启动和停止传输服务，Wire 生成的清理函数负责释放其余基础资源。

## 目录职责

```text
api/
  proto/common/v1/      Core 公共 protobuf 定义
  gen/go/common/v1/     protobuf 生成的 Go 代码

client/
  connection.go         远程或进程内 gRPC 连接适配
  localgrpc/             进程内 gRPC 服务注册与调用

audit/                   Core 审计事件契约、队列投递与异步入库
biz/                     基础上下文、认证授权和公共业务能力
config/                  启动配置解析
const/                   公共常量
data/                    多数据源客户端、事务和 Core 数据仓储
errorsx/                 统一错误构造
job/                     Cron 注册、持久化任务和运行时
mcp/                     MCP 服务与生命周期适配
module/                  宿主模块、资源和协议注册契约
queue/                   队列消息辅助能力与消费者生命周期
resource/                文档、I18n、迁移、OpenAPI 和启动资源同步
  biz/                    API、租户和 Casbin 资源同步业务
    dto/                  资源同步 DTO
  docs/                   项目文档注册与查询
    dto/                  项目文档查询 DTO
  i18n/                   国际化资源合并
  locale/                 语言标识解析
  migration/              数据库迁移
  openapi/                OpenAPI 注册、查询和 HTTP 挂载
    dto/                  OpenAPI 查询 DTO
server/                  HTTP、gRPC 和中间件
  middleware/             HTTP/gRPC 通用中间件
sse/                     SSE 流注册、传输和发布
internal/models/         Core 内部数据库模型

bootstrap.go             对外 ProviderSet 和应用生命周期装配
Makefile                 生成、格式化、测试和静态检查命令
```

## 开发命令

```bash
make tools     # 安装并锁定代码生成与格式化工具
make api       # 生成 api/gen/go
make wire      # 在 WIRE_DIR 指定的宿主目录生成 Wire 代码
make fmt       # goimports 格式化 Go 代码
make test      # 检查根、api、client 三个 Go 模块
make vet       # 检查根、api、client 三个 Go 模块
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

项目要求 Go `1.27.0`。`api` 和 `client` 是独立 Go 模块，修改它们时还应分别执行 `cd api && go test ./...`、`cd client && go test ./...`。修改公共模块契约后，应额外编译依赖 Core 的宿主项目。

客户端连接的独立说明见 [client/README.md](client/README.md)。
