# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` 是 Kratos 服務的通用執行時。宿主專案負責業務 Case、Service、API 和程序入口；Core 負責基礎設施、傳輸層、資源註冊以及應用程式生命週期。

Core 不是完整的業務範本。宿主透過一個 `module.Module` 將業務服務和建置期資源交給 Core，再由 `NewApp` 統一組裝並交由宿主啟動。

## Core 負責什麼

- 從 `bootstrap.Context` 解析資料庫、Redis、佇列、OSS、JWT、翻譯器和效能分析設定。
- 根據模組資源建立多資料源 GORM 用戶端、快取、佇列、OSS、翻譯器和共用的 `biz.BaseCase`。
- 按設定建立 HTTP、gRPC、MCP、SSE、佇列和持久化 Cron 執行時，並將宿主服務註冊到對應傳輸層。
- 組裝時依序執行資料庫遷移、OpenAPI 同步、租戶角色選單同步和 Casbin 策略重建。
- 統一組裝可選服務並交由 Kratos 管理生命週期；由 Wire 產生的清理函式釋放其餘基礎資源。

## 公開邊界

跨專案可以依賴的 Go 程式碼分為四個入口：

- 根套件提供 `ProviderSet` 和 `NewApp`。宿主通常只需要把 `ProviderSet` 放入自己的 Wire 圖。
- 根目錄的 `biz`、`config`、`const`、`data`、`errorsx`、`job`、`mcp`、`module`、`queue`、`resource`、`server` 和 `sse` 提供跨專案公開套件。
- `api` 是獨立 Go 模組，`api/proto` 保存 Core 的 protobuf 定義，`api/gen/go` 保存產生的 Go 類型。
- `client` 是獨立 Go 模組，提供基於 `kratos-kit` 設定的 gRPC 連線和行程內 gRPC 連線。

`internal/models` 下的程式碼只屬於 Core 實作，不是跨專案 API。Core 建立的快取、佇列、OSS、翻譯器和多資料源 GORM 用戶端會注入 `biz.BaseCase`，同時寫入 `kratos-kit/sdk.Runtime`，業務程式碼可按需從 BaseCase 或 SDK 取得。

## Wire 接入

宿主在自己的 Wire 組合根中提供一個實作 `module.Module` 的業務模組，並使用 Core 唯一公開的 `ProviderSet`：

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

Core 的 `ProviderSet` 彙總設定、基礎設施、模組資源、資料存取、資源同步和各協議執行時，並包含 `NewApp`。宿主只需補充自己的業務 Provider，並提供 `[]module.Module`，不需要重複加入 Core 的 ProviderSet。Core 根目錄不再維護 `wire.go` 或 `wire_gen.go`；宿主專案應使用自己的 Wire 指令產生組合根和 `wire_gen.go`，也可以使用 `make wire WIRE_DIR=<宿主 Wire 目錄>`。

## 模組契約

業務模組實作 `module.Module`。模組自己持有業務 Service，並在協議註冊方法中完成註冊：

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

| 方法 | 作用 |
| --- | --- |
| `RegisterGRPC` | 註冊產生的 gRPC Service。未設定 gRPC 服務時 Core 不會建立 gRPC Server。 |
| `RegisterHTTP` | 註冊 HTTP Service 和路由。未設定 HTTP 服務時 Core 不會建立 HTTP Server。 |
| `RegisterMCP` | 註冊 MCP 工具。MCP 可以獨立監聽，也可以掛載到 HTTP。 |
| `RegisterQueue` | 註冊佇列消費者；Core 同時註冊內建日誌和任務日誌消費者。 |
| `RegisterCron` | 註冊資料庫持久化任務執行器，通常呼叫 `server.RegisterTask`。回傳錯誤會中止組裝。 |
| `RegisterSSE` | 註冊業務 SSE 流，通常呼叫 `server.RegisterStream`。回傳錯誤會中止組裝。 |
| `Resources` | 回傳模型、遷移、OpenAPI、專案文件和 I18n 等靜態資源。 |

多個業務模組可以由宿主 Wire 組合根作為 `module.Module` 提供。Core 會依提供順序收集資源並轉發協議註冊；重複文件路徑、衝突的 OpenAPI 文件、內容不同的 I18n 訊息鍵或重複 SSE 流標識會在組裝時被拒絕。

## 建置期資源

`module.Resources` 是每個模組提供的一次性資源快照：

| 欄位 | 內容與約束 |
| --- | --- |
| `ProjectKey` | 專案穩定標識，用於文件和 OpenAPI 命名；留空時使用 `kratos-core`。 |
| `ProjectName` | 專案展示名稱；留空時回退到 `ProjectKey`。 |
| `Models` | 按資料源名稱分組的 GORM 模型。含模型的資料源必須在設定中存在，預設資料源必須配置。 |
| `Migrations` | 版本化遷移列表。每個 `module.Migration` 宣告 `Name`、`FS`、`Path` 和 `Dependencies`，Core 依賴關係順序執行。 |
| `OpenAPI` | 包含 `openapi.yaml`、`openapi.yml` 或 `openapi.json`，以及可選的 `openapi.<locale>.yaml` 等語言檔案的 `fs.FS`。啟用 Swagger 後，Core 會按專案和語言掛載原文和 Swagger UI。 |
| `Docs` | 通常包含 `docs.json` 的 `fs.FS`；文件翻譯由產生器寫入 `locale`，依請求語言查詢並回退到預設正文。 |
| `I18n` | 包含 `zh-CN.json`、`zh-TW.json`、`en-US.json`、`ja-JP.json` 等語言檔案的 `fs.FS`。Core 會與內建文案合併。 |

宿主通常透過 `embed.FS`、程式碼產生器或 `fstest.MapFS` 提供這些資源：

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

## 執行時能力

### 共用上下文

`biz.BaseCase` 是宿主業務共用的基礎上下文，包含 `bootstrap.Context`、快取、佇列、OSS、翻譯器和多資料源 GORM 用戶端，並提供 `GetAuthInfo` 讀取目前認證使用者。

Core 還向宿主提供以下具體業務服務：

- `job.Job`：啟動、停止或立即執行資料庫中的持久化任務。
- `resource/docs.Docs`：查詢合併後的專案文件樹和依請求語言選擇的文件正文。
- `resource/openapi.OpenAPI`：按請求語言、服務或 HTTP 操作查詢 OpenAPI 資訊。
- `sse.SSE`：建立 SSE 訂閱並發布 JSON 事件。

### 服務與中介軟體

HTTP 和 gRPC 服務會按設定掛載 request ID、I18n、日誌、認證授權和參數驗證中介軟體。HTTP 也支援本地 OSS 靜態檔案、SPA 回退和 Swagger。行程內 MCP 或 SSE 端點會掛載到 HTTP 服務，因此使用任一模式時都必須設定 HTTP。

佇列執行時負責消費 Core 的日誌訊息和任務日誌訊息，並轉發宿主註冊的消費者。Cron 執行時從資料庫重新載入啟用的 `BaseJob`，執行模組在 `RegisterCron` 中註冊的處理器。

## 組裝與啟動順序

`ProviderSet` 與 `NewApp` 的主要組裝順序如下：

1. 解析啟動設定並收集模組資源，根據模組模型建立資料源和遷移註冊表。
2. 執行資料庫遷移，接著在同一事務中同步 OpenAPI 介面、`base_api_i18n` 語言快照、租戶角色選單和 Casbin 資料庫規則；介面語言記錄使用 `operation + locale` 唯一鍵，不關聯會變動的 `base_api.id`；事務提交後刷新記憶體策略。
3. 建立共用基礎服務、認證授權、HTTP/gRPC/MCP/SSE、佇列和 Cron 執行時，並呼叫模組註冊方法。
4. 組裝 Kratos App；Kratos 統一啟動和停止傳輸服務，由 Wire 產生的清理函式負責釋放其餘基礎設施。

## 目錄職責

```text
api/
  proto/common/v1/      Core 公共 protobuf 定義
  gen/go/common/v1/     protobuf 產生的 Go 程式碼

client/
  connection.go         遠端或行程內 gRPC 連線適配
  localgrpc/             行程內 gRPC 服務註冊與呼叫

biz/                     基礎上下文、認證授權和公開執行時業務能力
config/                  啟動設定解析
const/                   公共常數
data/                    多資料源用戶端、事務和 Core 資料儲存庫
errorsx/                 統一錯誤建構
job/                     Cron 註冊、持久化任務和執行時
mcp/                     MCP 服務與生命週期適配
module/                  宿主模組、資源和協議註冊契約
queue/                   佇列訊息輔助能力與消費者生命週期
resource/                文件、I18n、遷移、OpenAPI 和啟動資源同步
  biz/                    API、租戶和 Casbin 資源同步業務
    dto/                  資源同步 DTO
  docs/                   專案文件註冊與查詢
    dto/                  專案文件查詢 DTO
  i18n/                   國際化資源合併
  locale/                 語言標識解析
  migration/              資料庫遷移
  openapi/                OpenAPI 註冊、查詢和 HTTP 掛載
    dto/                  OpenAPI 查詢 DTO
server/                  HTTP、gRPC 和中介軟體
  middleware/             HTTP/gRPC 共用中介軟體
sse/                     SSE 流註冊、傳輸和發布
internal/models/         Core 內部資料庫模型

bootstrap.go             公開 ProviderSet 和應用程式生命週期組裝
Makefile                 產生、格式化、測試和靜態檢查命令
```

## 開發指令

```bash
make tools     # 安裝並鎖定程式碼產生與格式化工具
make api       # 產生 api/gen/go
make wire      # 在 WIRE_DIR 指定的宿主目錄產生 Wire 程式碼
make fmt       # 使用 goimports 格式化 Go 程式碼
make test      # 檢查根、api、client 三個 Go 模組
make vet       # 檢查根、api、client 三個 Go 模組
make lint      # 目前等同於 make vet
```

## 發布版本 tag

`scripts/tag_release.py` 參照 `kratos-kit`，按 Go module 獨立發布版本。執行前請先提交並推送程式碼；腳本只檢查遠端預設分支上的提交，不會提交工作區改動：

```bash
git add -A
git commit -m "提交說明"
git push origin main

make tag
```

預設掃描根模組、`api` 和 `client`，只有模組自上一個 tag 後存在已推送的程式碼更新時，才會建立並推送下一個 patch tag：

| 模組 | tag 格式 |
| --- | --- |
| 根模組 | `vX.Y.Z` |
| `api` | `api/vX.Y.Z` |
| `client` | `client/vX.Y.Z` |

也可以只處理指定模組：

```bash
MODULE=api make tag              # 從 api 目錄開始遞迴掃描
MODULE=api EXACT=1 make tag      # 只處理 api 模組
```

腳本會自動跳過沒有程式碼更新或 tag 已存在的模組；根模組的變更檢測會排除 `api` 和 `client` 子模組，避免子模組改動重複觸發根模組 tag。

專案要求 Go `1.27.0`。`api` 和 `client` 是獨立 Go 模組，修改它們時還應分別執行 `cd api && go test ./...`、`cd client && go test ./...`。修改公共模組契約後，還應額外編譯依賴 Core 的宿主專案。

客戶端連線的獨立說明見 [client/README.md](client/README.md)。
