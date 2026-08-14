# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md)

`kratos-core` is the base runtime for Kratos services. The host owns its business services and process entry point; Core owns configuration, infrastructure components, HTTP/gRPC/MCP/SSE assembly, resource synchronization, and the shared lifecycle.

## Public Boundary

Core keeps its public Go surface in three layers:

- The root package contains only `wire.go`, which exposes the root provider set, SDK runtime, and application constructor required by host Wire graphs.
- `pkg` contains only the five public packages `assets`, `biz`, `const`, `errorsx`, and `module`.
- `api/gen/go` contains protobuf types shared across projects.

All other implementations live under `internal` and are not cross-project APIs. Core creates the cache, queue, default database, OSS, and translator and stores them in `kratos-kit/sdk.Runtime`; host business code retrieves them directly from the SDK.

## Wire Integration

The host uses `ProviderSet` to initialize the Core root and inject each business capability directly, while `biz.ProviderSet` constructs the shared `BaseCase`:

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

`newHostModuleResources` returns `module.Resources`. Core first collects models, migrations, OpenAPI, project documentation, and i18n resources, initializes infrastructure, and stores it in `sdk.Runtime`. Wire then builds host cases and services before passing the complete `module.Module` to Core for startup.

## Module Contract

A business module implements `pkg/module.Module`; protocol registration remains owned by the module:

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

`module.Resources` contains static contributions needed before business construction: models, migrations, OpenAPI, project documentation, and i18n. `module.Contributions` contains only runtime task definitions, SSE streams, and the AI registration function. Core creates and owns the HTTP, gRPC, MCP, scheduled-task services, and middleware. The host receives `module.AIRegistrar` through `Contributions.AI` to register tools and fixed flows.

`pkg/module` describes what a host provides to Core and contains the AI registration and runtime contracts actually used by Admin. Shared AI types are defined directly in `pkg/module`; Core `internal/agent` only implements or consumes them, and `pkg/module` must not reference `internal`. Like `pkg/biz.Docs`, `pkg/biz.AI` is read-only and exposes only `Tools` and `FixedFlowProviders`; `biz.ProviderSet` constructs only `BaseCase`.

## Build-Time Resources

The host supplies build-time resources directly through `module.Resources`:

| Field | Purpose |
| --- | --- |
| `OpenAPI` | An OpenAPI YAML/JSON file system. |
| `Docs` | Generated project documentation, normally including `docs.json`. |
| `I18n` | A host locale JSON file system. |

Database migrations are supplied through `module.Resources.Migrations`. Each `module.Migration` explicitly declares its module name, file system, version-directory root path, and module dependencies.

The host supplies only `fs.FS` values for OpenAPI, Docs, and i18n. Core reads them into internal registries and the data structures required by `pkg/biz`; the i18n catalog lives in `internal/i18n`, while locale state and request-context handling live in `internal/locale`.

## Directory Responsibilities

```text
api/
  proto/                 Public Core protobuf definitions
  gen/go/                Generated protobuf Go code

pkg/
  assets/                Host build-time resources and document DTOs
  biz/                   Core-to-host capabilities and BaseCase Wire provider
  const/                 Public constants
  errorsx/               Unified error constructors
  module/                Host-to-Core resources, protocol registration, AI, and runtime contribution contracts

internal/
  application/           Application assembly, module collection, and lifecycle
  runtime/               Cache, queue, database, auth, OSS, and AI initialization
  data/                  Minimal Core models and repositories
  server/                HTTP, gRPC, MCP, and SSE assembly
  agent/, task/, job/    Internal AI, task, and Cron implementations
  docs/, i18n/, locale/, openapi/ Internal documentation, i18n, locale state, and OpenAPI implementations

wire.go                  Public Root, Runtime, and application Wire facade
```

## Development Commands

```bash
make api
make fmt
make test
make vet
make lint
```

The project requires Go `1.26.5`. Changes to public contracts must also be compile-checked against `kratos-admin` and `kratos-shop`.
