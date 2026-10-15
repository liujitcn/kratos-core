# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` is the shared runtime for Kratos services. It assembles infrastructure, protocol servers, resource synchronization, and application lifecycle. A business project owns its cases, services, APIs, models, and process entry point. Core documents and implements the capabilities in this repository and contains no concrete business module.

## Capabilities

- Parse startup settings and create multi-database clients, cache, queue, object storage, translators, authentication, authorization, and profiling runtimes.
- Register HTTP, gRPC, and MCP services through `module.Module`; inject jobs, queue consumers, and SSE streams through `job.Tasks`, `queue.Consumers`, and `sse.Streams`.
- Create optional HTTP, gRPC, MCP, SSE, queue, and persistent Cron services and let Kratos manage their lifecycle and errors.
- Run database migrations, OpenAPI synchronization, API localization, tenant role-menu synchronization, and policy rebuilding during assembly.
- Provide shared business context, transaction boundaries, log pipeline, error reasons, OpenAPI queries, SSE publishing, and persistent job controls.
- Provide the public protobuf module in `api` and the gRPC client module in `client`, including remote, discovery, and in-process calls.

## Public Packages

| Package | Responsibility |
| --- | --- |
| Root package | Exposes `ProviderSet` and `NewApp` for configuration, resources, data access, and protocol runtimes. |
| `biz` | Shared `BaseCase`, authentication and authorization, log pipeline, infrastructure clients, and runtime context. |
| `config` | Parses application, database, Redis, queue, object storage, JWT, translator, and profiling settings. |
| `data` | Creates multi-database clients and defines storage contracts for APIs, jobs, logs, permissions, and transactions. |
| `errorsx` | Builds consistent invalid-argument, authentication, permission, conflict, rate-limit, and internal errors. |
| `job` / `queue` / `sse` / `mcp` | Registration, runtime, and lifecycle adapters for persistent jobs, queues, SSE, and MCP. |
| `module` / `resource` | Module contracts, models, migrations, OpenAPI, localization, and startup synchronization. |
| `server` | HTTP and gRPC servers plus authentication, tenant, validation, logging, rate-limit, and policy middleware. |
| `api` | Independent Go module containing common protobuf definitions and generated code. |
| `client` | Independent Go module for remote, discovery, and in-process gRPC connections. |

## Integration

The root of Core does not maintain a business project's Wire composition root. The business project adds `core.ProviderSet` to its own Wire graph and supplies:

1. One or more `module.Module` values for HTTP, gRPC, and MCP registration.
2. `module.Resource` values for models, migrations, OpenAPI, and localization file systems.
3. Optional `job.Tasks`, `queue.Consumers`, and `sse.Streams` collections; use empty collections when a capability is not needed.
4. Implementations of the storage interfaces declared by `data` for startup synchronization, jobs, logs, and permissions.

`NewApp` collects these dependencies and creates the Kratos application. HTTP, gRPC, standalone MCP, standalone SSE, queue, and Cron are optional. No listener is created when the corresponding configuration is absent; an empty collection does not prevent a service from starting. In-process MCP and SSE are hosted by HTTP, so HTTP must be configured for those modes.

## Module Contract

The current module contract has three protocol registration methods:

```go
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*http.Server)
	RegisterMCP(*mcp.Server)
}
```

Core forwards registration in module order. A module may register only the protocols it uses. OpenAPI documents, localization keys, and SSE stream IDs must remain unique across modules; conflicts fail application assembly.

## Build-Time Resources

Business modules implement `module.Resource`:

| Method | Contract |
| --- | --- |
| `ProjectKey()` / `ProjectName()` | Stable project identifier and display name; empty values fall back to `kratos-core` and the identifier. |
| `Models()` | GORM models grouped by data-source name. Every model data source and the default data source must be configured. |
| `Migrations()` | Versioned migrations with a stable name, file system root, and dependencies. |
| `OpenAPI()` | An `fs.FS` containing `openapi.yaml`, `openapi.yml`, or `openapi.json`, optionally with locale variants. |
| `I18n()` | JSON message files named with locale identifiers. Conflicting duplicate keys are rejected. |

Migration resources follow this layout and run in dependency order:

```text
assets/<version>/<database-type>/README.md
assets/<version>/<database-type>/README.<locale>.md
assets/<version>/<database-type>/<data-source>/README.md
assets/<version>/<database-type>/<data-source>/README.<locale>.md
```

Migration records store file references and SHA-256 digests. `resource/migration.Migration.ReadFiles` preserves file order and boundaries while validating resource ownership, data source, file existence, and content digests. SQL digests use the content after execution-time comment removal. Historical migration files must remain available and should not be exposed as public static files.

## Runtime Capabilities

### Shared Context and Data

`biz.BaseCase` contains the startup context, cache, queue, object storage, translator, and multi-database clients, and exposes `GetAuthInfo` for the current authenticated principal. `data.Transaction` defines the transaction boundary. Business projects implement API, job, log, and permission stores; Core depends only on those contracts.

Core also provides consistent `errorsx` reasons, the asynchronous `biz.LogPipeline`, and common protobuf messages under `api/common/v1` for status, errors, analytics, options, array wrappers, and secret transport.

### HTTP and gRPC

Configured services can add request ID, localization, logging, authentication and authorization, tenant scope, validation, and rate-limit middleware. HTTP also supports:

- Read-only local object storage at `/data/`, with a configurable root directory.
- SPA directories under `web`, falling back to `index.html` when a file is not found.
- OpenAPI and Swagger at `/api/docs/openapi` and `/api/docs/swagger`.
- In-process MCP and SSE routes, with configurable paths.

HTTP, gRPC, and MCP constructors accept a policy resolver. Core puts the resolver into request context so business wrappers can read the request policy; Core does not modify business responses a second time. Cancellation and deadlines remain attached to the request context.

### MCP

Modules register tools through `RegisterMCP`. MCP supports standalone HTTP, Legacy SSE, STDIO, and in-process mounting on the Core HTTP server. In-process mode reuses the HTTP listener; standalone mode has its own Kratos-managed lifecycle.

### SSE

`sse.Streams` declare business streams. Core resolves a transport stream from `stream`, `channel`, user, and tenant information. Requests validate the Bearer token, role, tenant scope, and cached token validity. `sse.SSE` exposes `Serve`, `PublishJSON`, and `PublishEnabled`. HTTP subscriptions remove the service deadline while preserving cancellation when the client disconnects.

### Queue

`queue.Server` handles Core logs, job logs, and business consumers. Use `queue.AddQueue` for JSON messages, `ScheduleQueue` and `CancelQueue` for delayed messages, and `queue.Decode[T]` to validate and decode the `data` field.

### Persistent Jobs

`job.Tasks` registers task executors. The scheduler reloads enabled `BaseJob` records at startup. `job.Job` exposes `Start`, `Stop`, and `Run` for scheduling, removal, and immediate execution. Redis is preferred for execution locks; an in-process lock is used as a fallback and guarantees mutual exclusion only within one instance. Startup failures clean up registered jobs and execution context.

### OpenAPI, Localization, and Synchronization

The OpenAPI registry stores documents by project, locale, and operation. Duplicate keys are accepted only when name and content match; duplicate operations in one locale are rejected. Queries can fall back from the request locale to the default document and return the document or schema tree for an operation.

`resource/openapi.OpenAPI` is the runtime entry point for queries by request locale, service name, HTTP path, and method.

The localization catalog merges Core and module JSON files and rejects conflicting duplicate keys. After migrations, startup synchronization converts OpenAPI into API permission and localization snapshots, copies the default tenant administrator menu to ordinary tenant role copies, rebuilds database policies, and refreshes in-memory policies.

## Assembly Order

1. Parse startup settings, collect module resources, and create database clients by data source.
2. Register and run migrations, validating files and dependency order.
3. Synchronize OpenAPI, localization, tenant role menus, and policies in a transaction; refresh in-memory policies after commit.
4. Create shared context, authentication and authorization, protocol servers, queue, and Cron, then register modules and independent collections.
5. Assemble the Kratos App. Kratos starts and stops transport servers; Wire-generated cleanup releases the remaining infrastructure.

## Directory Responsibilities

```text
api/                     Common protobuf definitions and generated code
client/                  gRPC clients, client middleware, and in-process connection
biz/                     BaseCase, auth, log pipeline, and shared business runtime
config/                  Startup configuration parsing
const/                   Public constants
data/                    Multi-database clients, transactions, and storage contracts
errorsx/                 Consistent error construction
job/                     Cron registration, persistent jobs, and lifecycle
mcp/                     MCP service and lifecycle adapter
module/                  Module, resource, and protocol contracts
queue/                   Queue publishing, decoding, and consumer lifecycle
resource/                OpenAPI, localization, migrations, and startup sync
server/                  HTTP, gRPC, request metadata, and middleware
sse/                     SSE stream registration, subscriptions, and publishing
bootstrap.go             ProviderSet and application assembly
Makefile                 Generation, formatting, testing, checks, and version scripts
```

## Development Commands

The project requires Go `1.27.0`. The root, `api`, and `client` directories are independent Go modules; run the relevant tests after changing any module.

```bash
make init                         # Install generation, formatting, and Wire tools
make api                          # Generate api/gen/go from proto files
make wire WIRE_DIR=<directory>    # Generate Wire code in a composition root
make fmt                          # Format Go code and imports
make test                         # Test the root, api, and client modules
make vet                          # Run go vet for all three modules
make lint                         # Alias for make vet
make buf-push                     # Generate and push the protobuf schema (login required)
make tag                          # Create patch tags for committed module changes
```

`make tag` handles the root, `api`, and `client` modules by default. Tags are `vX.Y.Z`, `api/vX.Y.Z`, and `client/vX.Y.Z`; use `MODULE=api make tag` to choose a starting module and `EXACT=1` to process only that module. The script never commits working-tree changes.

See [client/README.md](client/README.md) for the client connection guide.
