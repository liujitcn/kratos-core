# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` 是 Kratos 服务的通用运行时。它负责基础设施装配、协议服务、公共资源同步和应用生命周期；业务项目负责业务 Case、Service、API、模型以及进程入口。Core 只描述和实现本仓库提供的能力，不包含具体业务模块。

## 功能概览

- 解析启动配置，创建多数据源数据库客户端、缓存、队列、对象存储、翻译器、认证授权和性能分析运行时。
- 通过 `module.Module` 注册 HTTP、gRPC 和 MCP 服务，通过独立的 `job.Tasks`、`queue.Consumers`、`sse.Streams` 注入定时任务、队列消费者和 SSE 流。
- 按配置创建 HTTP、gRPC、MCP、SSE、队列和持久化 Cron 服务，并由 Kratos 统一启动、停止和传播服务错误。
- 在应用组装阶段执行数据库迁移、OpenAPI 接口同步、接口国际化、租户角色菜单同步和权限策略重建。
- 提供通用业务上下文、事务边界、日志流水线、错误原因、OpenAPI 查询、SSE 发布和持久化任务控制能力。
- 提供 `api` 公共 protobuf 模块和 `client` gRPC 客户端模块，支持远程、服务发现和进程内调用。

## 公共入口

| 入口 | 当前职责 |
| --- | --- |
| 根包 | 提供 `ProviderSet` 和 `NewApp`，汇总配置、资源、数据访问和协议运行时。 |
| `biz` | 提供 `BaseCase`、认证授权、日志流水线、基础设施客户端和运行时公共上下文。 |
| `config` | 从启动上下文解析应用、数据库、Redis、队列、对象存储、JWT、翻译器和性能分析配置。 |
| `data` | 创建多数据源客户端，并定义 API、任务、日志、权限和事务的存储契约。 |
| `errorsx` | 统一构造参数错误、认证失败、无权限、冲突、限流和内部错误。 |
| `job` / `queue` / `sse` / `mcp` | 提供持久化任务、队列、SSE 和 MCP 的注册、运行时与生命周期适配。 |
| `module` / `resource` | 定义业务模块、模型、迁移、OpenAPI、国际化资源以及启动期同步流程。 |
| `server` | 创建 HTTP、gRPC 服务和请求中间件，包括认证、租户范围、校验、日志、限流和策略上下文。 |
| `api` | 独立 Go 模块，保存公共 protobuf 定义和生成代码。 |
| `client` | 独立 Go 模块，统一创建远程、服务发现或进程内 gRPC 客户端连接。 |

## 接入方式

Core 根目录不维护业务项目的 Wire 组合根。业务项目在自己的 Wire 入口中注入 `core.ProviderSet`，并提供以下依赖：

1. 一个或多个 `module.Module`，用于注册 HTTP、gRPC 和 MCP 服务。
2. `module.Resource`，用于提供模型、迁移、OpenAPI 和国际化文件系统。
3. 可选的 `job.Tasks`、`queue.Consumers` 和 `sse.Streams` 集合；没有对应能力时提供空集合。
4. `data` 包声明的存储接口实现，用于启动同步、任务、日志和权限资源。

`NewApp` 会收集这些依赖并创建最终的 Kratos 应用。HTTP、gRPC、独立 MCP、独立 SSE、队列和 Cron 均为可选服务；没有对应配置时不会创建监听器，空集合不会阻止服务启动。进程内 MCP 和 SSE 由 HTTP 服务承载，因此启用进程内模式时必须同时配置 HTTP。

## 模块契约

业务模块只需要实现当前版本的三个协议注册方法：

```go
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*http.Server)
	RegisterMCP(*mcp.Server)
}
```

Core 按模块提供顺序转发注册调用。模块可以只实现实际使用的协议；未配置的协议不会创建服务。多个模块的 OpenAPI 文档、国际化键和 SSE 流标识必须保持唯一，冲突会在应用组装时返回错误。

## 构建期资源

`module.Resource` 由业务模块实现：

| 方法 | 约束 |
| --- | --- |
| `ProjectKey()` / `ProjectName()` | 项目标识和展示名称；为空时分别回退到 `kratos-core` 和项目标识。 |
| `Models()` | 按数据源名称返回 GORM 模型。使用模型的数据源必须存在，默认数据源必须配置。 |
| `Migrations()` | 返回带名称、文件系统根路径和依赖关系的版本化迁移集合。 |
| `OpenAPI()` | 返回包含 `openapi.yaml`、`openapi.yml` 或 `openapi.json` 的文件系统，可附带语言版本。 |
| `I18n()` | 返回以 locale 文件名保存的 JSON 文案文件系统。相同语言的重复键必须具有相同内容。 |

迁移资源使用以下目录约定，Core 按 `Dependencies` 拓扑顺序执行：

```text
assets/<version>/<database-type>/README.md
assets/<version>/<database-type>/README.<locale>.md
assets/<version>/<database-type>/<data-source>/README.md
assets/<version>/<database-type>/<data-source>/README.<locale>.md
```

迁移记录保存文件引用和 SHA-256 摘要。`resource/migration.Migration.ReadFiles` 会按记录顺序读取文件，校验路径归属、数据源、文件存在性和内容摘要；SQL 校验使用执行器去除注释后的内容。部署时必须保留历史迁移文件，迁移资源目录不应直接作为公开静态目录。

## 运行时能力

### 基础上下文与数据访问

`biz.BaseCase` 聚合启动上下文、缓存、队列、对象存储、翻译器和多数据源数据库客户端，并提供 `GetAuthInfo` 读取当前认证主体。`data.Transaction` 统一事务边界；API、任务、日志和权限存储由业务项目实现，Core 只依赖这些接口。

Core 还提供：

- `errorsx`：统一错误原因和消息包装，HTTP/gRPC 可映射为标准状态码。
- `biz.LogPipeline`：将 API 访问、策略评估和任务日志异步写入队列与存储，并在缓冲关闭或满载时返回明确错误。
- `api/common/v1`：状态、通用错误、分析图表、树形/下拉选项、数组包装和敏感字段传输消息。

### HTTP 与 gRPC

服务按配置挂载请求标识、国际化、日志、认证授权、租户范围、参数校验和接口限流中间件。HTTP 还支持：

- 本地对象存储只读目录 `/data/`；根目录可由对象存储配置覆盖。
- `web` 下按子目录注册 SPA，找不到静态文件时回退到 `index.html`。
- Swagger/OpenAPI 文档：`/api/docs/openapi` 和 `/api/docs/swagger`。
- 进程内 MCP 与 SSE 路由挂载；路径可由服务配置覆盖。

HTTP、gRPC 和 MCP 构造时可注入策略解析器。Core 将解析器放入请求上下文，业务包装器可以读取当前请求策略；Core 不重复修改业务响应。取消信号和 deadline 会随请求上下文传递。

### MCP

模块通过 `RegisterMCP` 注册工具。MCP 支持独立 HTTP、Legacy SSE、STDIO 和挂载到 Core HTTP 的进程内模式。进程内模式复用 HTTP 监听器，独立模式由 Kratos 单独管理生命周期。

### SSE

`sse.Streams` 注册业务流，Core 根据 `stream`、`channel`、用户和租户解析实际传输流。请求会校验 Bearer 令牌、用户角色、租户范围以及缓存中的令牌有效性；失效令牌不会因为同账号其他设备在线而恢复访问。`sse.SSE` 提供 `Serve` 建立订阅、`PublishJSON` 发布 JSON 事件和 `PublishEnabled` 状态查询。HTTP 订阅会移除服务级 deadline，但保留客户端断开产生的取消信号。

### 队列

`queue.Server` 管理 Core 日志、任务日志和业务消费者。业务代码可使用：

- `queue.AddQueue` 投递 JSON 消息。
- `queue.ScheduleQueue` 投递延迟消息，`queue.CancelQueue` 取消尚未触发的消息。
- `queue.Decode[T]` 从消息的 `data` 字段解码业务对象，并校验空消息和 JSON 格式。

### 持久化任务

`job.Tasks` 注册任务执行器；调度器启动时从存储重新加载启用的 `BaseJob`。`job.Job` 提供 `Start`、`Stop` 和 `Run`，分别用于创建调度入口、移除调度和立即执行一次任务。任务入口优先使用 Redis 分布式锁；Redis 不可用时降级为进程内锁，只保证单实例互斥。启动失败会清理已注册任务和执行上下文。

### OpenAPI、国际化与启动同步

OpenAPI 注册表按项目、语言和接口操作保存文档，重复 key 只允许名称和内容完全一致；同语言下重复接口会被拒绝。查询 API 支持按请求语言回退默认文档，并可返回接口对应文档和 schema 字段树。

`resource/openapi.OpenAPI` 提供按请求语言、服务名、HTTP 路径和方法查询文档的运行时入口。

国际化目录会合并 Core 与模块 JSON；同一语言文件出现内容不同的重复键时，应用组装失败。启动同步在迁移完成后执行：

1. 将 OpenAPI 文档转换为接口权限和接口国际化快照。
2. 同步租户管理员角色菜单到普通租户角色副本。
3. 根据角色、租户、菜单和接口重建数据库权限规则，并刷新内存策略。

## 组装与启动顺序

1. 解析启动配置，收集模块资源，按数据源创建数据库客户端。
2. 注册并执行迁移，校验迁移文件和依赖顺序。
3. 在事务中同步 OpenAPI、国际化、租户角色菜单和权限规则；提交后刷新内存策略。
4. 创建公共上下文、认证授权、协议服务、队列和 Cron，并注册模块及独立集合。
5. 组装 Kratos App。Kratos 管理传输服务的启动和停止，Wire 生成的清理函数释放其余基础资源。

## 目录职责

```text
api/                     公共 protobuf 定义和生成代码
client/                  gRPC 客户端、客户端中间件和进程内连接
biz/                     BaseCase、认证授权、日志流水线和公共业务能力
config/                  启动配置解析
const/                   公共常量
data/                    多数据源客户端、事务和存储契约
errorsx/                 统一错误构造
job/                     Cron 注册、持久化任务和调度生命周期
mcp/                     MCP 服务和生命周期适配
module/                  模块、资源和协议注册契约
queue/                   队列投递、解码和消费者生命周期
resource/                OpenAPI、国际化、迁移和启动同步
server/                  HTTP、gRPC、请求元数据和中间件
sse/                     SSE 流注册、订阅和发布
bootstrap.go             ProviderSet 和应用生命周期组装
Makefile                 生成、格式化、测试、检查和版本脚本入口
```

## 开发命令

项目要求 Go `1.27.0`。根模块、`api` 和 `client` 是三个独立 Go 模块，修改任一模块后都应执行对应测试。

```bash
make init                         # 安装生成、格式化和 Wire 工具
make api                          # 由 proto 生成 api/gen/go
make wire WIRE_DIR=<目录>         # 在指定组合根生成 Wire 代码
make fmt                          # 格式化 Go 源码和导入
make test                         # 测试根、api、client 三个模块
make vet                          # 对三个模块执行 go vet
make lint                         # 当前等同于 make vet
make buf-push                     # 生成并推送 protobuf schema（需登录）
make tag                          # 按模块创建已提交代码的 patch tag
```

`make tag` 默认处理根模块、`api` 和 `client`：根模块使用 `vX.Y.Z`，子模块分别使用 `api/vX.Y.Z` 和 `client/vX.Y.Z`。可用 `MODULE=api make tag` 指定起始模块，或使用 `EXACT=1` 只处理该模块。脚本不会提交工作区改动。

客户端连接的独立说明见 [client/README.md](client/README.md)。
