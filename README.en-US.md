# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` is the shared runtime for Kratos services. The host project owns its business cases, services, APIs, and process entry point; Core owns infrastructure, transports, resource registration, and application lifecycle.

Core is not a complete business template. A host supplies one `module.Module` containing its business services and build-time resources, and `NewApp` assembles and starts the application.

## Core Responsibilities

- Parse database, Redis, queue, OSS, JWT, translator, and profiling settings from `bootstrap.Context`.
- Build multi-database GORM clients, cache, queue, OSS, translator, and the shared `biz.BaseCase` from module resources.
- Create HTTP, gRPC, MCP, SSE, queue, and persistent Cron runtimes according to configuration, then register host services with the matching transport.
- Run database migrations, OpenAPI synchronization, tenant role/menu synchronization, and Casbin policy rebuilds during startup.
- Start and stop Kratos services as one application and return an application cleanup function.

## Public Boundary

Cross-project Go code is exposed through four entry points:

- The root package provides `ProviderSet` and `NewApp`. A host normally only adds `ProviderSet` to its own Wire graph.
- `pkg` provides the public `biz`, `config`, `const`, `dto`, `errorsx`, and `module` packages.
- `api` is an independent Go module. `api/proto` contains Core protobuf definitions and `api/gen/go` contains generated Go types.
- `client` is an independent Go module that provides gRPC connections based on `kratos-kit` configuration, including in-process gRPC connections.

Everything under `internal` is an implementation detail and is not a cross-project API. Core-created cache, queue, OSS, translator, and multi-database GORM clients are injected into `biz.BaseCase` and also stored in `kratos-kit/sdk.Runtime`; host code can use either BaseCase or the SDK as needed.

## Wire Integration

The host provides a business module implementing `module.Module` and uses Core's only public Wire entry point:

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
		wire.Bind(new(module.Module), new(*hostModule)),
	))
}
```

Core's `ProviderSet` only adds `NewApp` to the host's Wire graph; it does not expose `BaseCase`, `Job`, `Docs`, `OpenAPI`, or `SSE` as host Wire outputs. Host business cases should still add the public `biz.ProviderSet`, `config.ProviderSet`, and their own providers as needed. `wire_gen.go` must be generated with `make wire` or the host project's Wire command; it is not hand-maintained.

## Module Contract

The business module implements `module.Module`. It owns its business services and registers them in the protocol methods:

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

| Method | Responsibility |
| --- | --- |
| `RegisterGRPC` | Register generated gRPC services. Core skips the gRPC server when it is not configured. |
| `RegisterHTTP` | Register HTTP services and routes. Core skips the HTTP server when it is not configured. |
| `RegisterMCP` | Register MCP tools. MCP can listen independently or be mounted on HTTP. |
| `RegisterQueue` | Register queue consumers; Core also registers built-in log and job-log consumers. |
| `RegisterCron` | Register persistent database task executors, usually with `server.RegisterTask`. An error aborts assembly. |
| `RegisterSSE` | Register business SSE streams, usually with `server.RegisterStream`. An error aborts assembly. |
| `Resources` | Return models, migrations, OpenAPI, project documentation, and i18n resources. |

Multiple business modules can be passed as separate arguments to `NewApp`. Core collects resources and forwards protocol registration in argument order. Duplicate documentation paths, conflicting OpenAPI documents, i18n keys with different contents, or duplicate SSE stream IDs are rejected during assembly.

## Build-Time Resources

`module.Resources` is a one-time resource snapshot supplied by each module:

| Field | Contents and constraints |
| --- | --- |
| `ProjectKey` | Stable project identifier used for documentation and OpenAPI namespacing; defaults to `kratos-core`. |
| `ProjectName` | Display name for the project; falls back to `ProjectKey`. |
| `Models` | GORM models grouped by data-source name. Every data source containing models must be configured, and the default data source is required. |
| `Migrations` | Versioned migrations. Each `module.Migration` declares `Name`, `FS`, `Path`, and `Dependencies`; Core runs them in dependency order. |
| `OpenAPI` | An `fs.FS` containing `openapi.yaml`, `openapi.yml`, or `openapi.json`, plus optional `openapi.<locale>.yaml` language files. When Swagger is enabled, Core mounts raw documents and Swagger UI by project and locale. |
| `Docs` | An `fs.FS` normally containing `docs.json`; the generator writes translations into `locale`, which are selected by request locale and fall back to the default body. |
| `I18n` | An `fs.FS` containing locale files such as `zh-CN.json`, `zh-TW.json`, `en-US.json`, and `ja-JP.json`; Core merges them with its built-in messages. |

Hosts commonly provide these resources through `embed.FS`, a code generator, or `fstest.MapFS`:

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

## Runtime Capabilities

### Shared Context

`biz.BaseCase` is the shared host business context. It contains `bootstrap.Context`, cache, queue, OSS, translator, and multi-database GORM clients, and provides `GetAuthInfo` for reading the authenticated user.

Core also provides these concrete runtime services:

- `job.Job`: start, stop, or immediately run persistent database jobs.
- `resource/docs.Docs`: query the merged project documentation tree and document bodies selected by request locale.
- `resource/openapi.OpenAPI`: query OpenAPI information by request locale, service, or HTTP operation.
- `sse.SSE`: create SSE subscriptions and publish JSON events.

### Services and Middleware

HTTP and gRPC services add request ID, i18n, logging, authentication/authorization, and validation middleware according to configuration. HTTP also supports local OSS static files, SPA fallback, and Swagger. In-process MCP or SSE endpoints are mounted on the HTTP server, so HTTP must be configured when either mode is used.

The queue runtime consumes Core log and job-log messages and forwards host consumers. The Cron runtime reloads enabled `BaseJob` records from the database and executes handlers registered by modules in `RegisterCron`.

## Startup Order

The main `NewApp` assembly sequence is:

1. Parse startup settings and collect module resources, then create data sources and the migration registry from module models.
2. Run database migrations, then synchronize OpenAPI APIs, the `base_api_i18n` locale snapshot, tenant role menus, and Casbin database rules in one transaction; locale rows use the `operation + locale` key and do not reference the mutable `base_api.id`; refresh in-memory policies after commit.
3. Create shared infrastructure, authentication/authorization, HTTP/gRPC/MCP/SSE, queue, and Cron runtimes, invoking module registration methods.
4. Assemble the Kratos App. Kratos owns transport startup and shutdown; the returned cleanup function releases the remaining infrastructure.

## Directory Responsibilities

```text
api/
  proto/common/v1/      Core protobuf definitions
  gen/go/common/v1/     Generated protobuf Go code

client/
  connection.go         Remote or in-process gRPC connection adapter
  localgrpc/             In-process gRPC registration and invocation

biz/                     Shared context and Core built-in business cases
biz/dto/                 OpenAPI parsing DTOs
config/                  Startup configuration parsing
const/                   Public constants
internal/data/           Core models, transactions, and repositories (Core-internal)
docs/dto/                Project documentation query DTOs
errorsx/                 Unified error constructors
job/                     Cron registration, persistent jobs, and runtime
mcp/                     MCP service and lifecycle adapter
module/                  Host module, resource, and protocol contracts
openapi/dto/             OpenAPI query DTOs
queue/                   Queue consumers and lifecycle adapter
resource/                Documentation, i18n, migration, OpenAPI, and startup sync
server/                  HTTP, gRPC, MCP, and middleware
sse/                     SSE stream registration, transport, and publishing

bootstrap.go             Public ProviderSet and application lifecycle assembly
wire.go                  Core Wire composition root
wire_gen.go              Wire-generated output
Makefile                 Generation, formatting, testing, and static checks
```

## Development Commands

```bash
make api       # Generate api/gen/go
make wire      # Generate wire_gen.go
make fmt       # Format Go code with goimports
make test      # go test ./...
make vet       # go vet ./...
make lint      # Currently equivalent to make vet
```

The project requires Go `1.26.5`. `api` and `client` are independent Go modules; when changing either one, also run `cd api && go test ./...` or `cd client && go test ./...`. Changes to public module contracts should additionally be compile-checked against the host projects that depend on Core.

See [client/README.md](client/README.md) for the standalone client connection guide.
