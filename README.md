# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md)

`kratos-core` 是 Kratos 服务的基础运行时。宿主负责业务服务和进程入口，Core 负责配置、基础组件、HTTP/gRPC/MCP/SSE、资源同步以及统一生命周期。

## 公共边界

Core 的公开 Go 代码保持为三个层次：

- 根包仅用 `wire.go` 提供宿主 Wire 图需要的根 Provider、SDK Runtime 和应用构造函数。
- `pkg` 仅保留 `assets`、`biz`、`const`、`errorsx`、`module` 五个公共包。
- `api/gen/go` 提供跨项目共享的 protobuf 类型。

其他实现全部位于 `internal`，不属于跨项目 API。缓存、队列、默认数据库、OSS 和翻译器由 Core 创建后写入 `kratos-kit/sdk.Runtime`，业务代码直接从 SDK 获取，不再经过 Core 二次封装。

## Wire 接入

宿主在自己的 Wire 组合根中使用 `ProviderSet` 初始化 Core 根对象并逐项注入业务能力，同时使用 `biz.ProviderSet` 构建共用的 `BaseCase`：

```go
//go:build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	core "github.com/liujitcn/kratos-core"
	"github.com/liujitcn/kratos-core/pkg/biz"
	"github.com/liujitcn/kratos-core/pkg/module"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

func initializeApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		core.ProviderSet,
		biz.ProviderSet,
		newHostModuleResources,
		newHostModule,
		wire.Bind(new(module.Module), new(*hostModule)),
	))
}
```

`newHostModuleResources` 返回 `module.Resources`。Core 先收集其中的模型、迁移、OpenAPI、项目文档和 I18n，初始化数据库、缓存、队列等基础设施并写入 `sdk.Runtime`；Wire 随后构建宿主 Case 和 Service，最终把完整 `module.Module` 交给 Core 启动。

## 模块契约

业务模块实现 `pkg/module.Module`，服务注册仍由模块自己负责：

```go
type hostModule struct{}

func (*hostModule) RegisterGRPC(grpc.ServiceRegistrar) {}
func (*hostModule) RegisterHTTP(*kratosHTTP.Server)    {}
func (*hostModule) RegisterMCP(*mcpserver.Server)      {}
func (m *hostModule) Contributions() module.Contributions {
	return module.Contributions{AI: m.registerAI}
}
func (*hostModule) registerAI(module.AIRegistrar) error { return nil }
```

`module.Resources` 是业务构建前的静态贡献，包含模型、迁移、OpenAPI、项目文档和 I18n；`module.Contributions` 是业务构建后的运行期贡献，只包含任务、SSE 和 AI 注册函数。HTTP、gRPC、MCP、定时任务等服务以及拦截器均由 Core 统一创建和管理。宿主通过 `Contributions.AI` 接收 `module.AIRegistrar` 并注册工具与固定流程。

`pkg/module` 描述宿主向 Core 提供的内容，并集中保存 Admin 实际使用的 AI 注册与运行契约；这些共享类型直接定义在 `pkg/module`，Core `internal/agent` 只实现或消费它们，不允许 `pkg/module` 反向引用 `internal`。`pkg/biz.AI` 和 `pkg/biz.Docs` 一样只提供只读查询，其中 AI 仅暴露 `Tools` 与 `FixedFlowProviders`；`biz.ProviderSet` 只构建 `BaseCase`。

## 构建期资源

宿主通过 `module.Resources` 直接提供构建期资源：

| 字段 | 用途 |
| --- | --- |
| `OpenAPI` | OpenAPI YAML/JSON 文件系统。 |
| `Docs` | 项目文档生成结果，通常包含 `docs.json`。 |
| `I18n` | 宿主语言 JSON 文件系统。 |

数据库迁移通过 `module.Resources.Migrations` 提供，每项 `module.Migration` 显式声明模块名、文件系统、版本目录根路径和依赖模块。

宿主只为 OpenAPI、Docs 和 I18n 提供 `fs.FS`。Core 读取后转换为内部注册表和 `pkg/biz` 查询所需的数据结构；国际化目录实现在 `internal/i18n`，语言状态和上下文实现在 `internal/locale`。

## 目录职责

```text
api/
  proto/                 Core 公共 protobuf 定义
  gen/go/                protobuf 生成的 Go 代码

pkg/
  assets/                宿主构建期资源和文档 DTO
  biz/                   Core 向宿主提供的业务能力及 BaseCase Wire Provider
  const/                 公共常量
  errorsx/               统一错误构造
  module/                宿主向 Core 提供的资源、协议注册、AI 与运行期贡献契约

internal/
  application/           应用装配、模块资源收集和统一生命周期
  runtime/               cache、queue、database、auth、OSS 和 AI 初始化
  data/                  Core 最小模型和仓储
  server/                HTTP、gRPC、MCP、SSE 服务装配
  agent/, task/, job/    AI、任务和 Cron 内部实现
  docs/, i18n/, locale/, openapi/ 文档、国际化、语言状态和 OpenAPI 内部实现

wire.go                  对宿主公开的 Root、Runtime 和应用 Wire 门面
```

## 开发命令

```bash
make api
make fmt
make test
make vet
make lint
```

项目要求 Go `1.26.5`。修改公共契约后需要同时验证 `kratos-admin` 和 `kratos-shop` 的编译。
