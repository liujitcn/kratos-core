# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` 是 Kratos 服務的通用執行時。它負責基礎設施組裝、協定服務、公共資源同步與應用程式生命週期；業務專案負責業務 Case、Service、API、模型與程序入口。Core 只描述本儲存庫提供的能力，不包含具體業務模組。

## 功能概覽

- 解析啟動設定，建立多資料源資料庫用戶端、快取、佇列、物件儲存、翻譯器、認證授權與效能分析執行時。
- 透過 `module.Module` 註冊 HTTP、gRPC、MCP，透過 `job.Tasks`、`queue.Consumers`、`sse.Streams` 注入定時任務、佇列消費者與 SSE 流。
- 按設定建立 HTTP、gRPC、MCP、SSE、佇列和持久化 Cron 服務，由 Kratos 統一管理啟停與錯誤。
- 在組裝階段執行資料庫遷移、OpenAPI 同步、介面國際化、租戶角色選單同步與權限策略重建。
- 提供共用業務上下文、交易邊界、日誌流水線、錯誤原因、OpenAPI 查詢、SSE 發布與持久化任務控制。
- 提供 `api` 公共 protobuf 模組和 `client` gRPC 用戶端模組，支援遠端、服務發現與行程內呼叫。

## 公開入口

| 入口 | 職責 |
| --- | --- |
| 根套件 | 提供 `ProviderSet`、`NewApp`，彙總設定、資源、資料存取與協定執行時。 |
| `biz` | `BaseCase`、認證授權、日誌流水線、基礎設施用戶端與共用上下文。 |
| `config` | 解析應用、資料庫、Redis、佇列、物件儲存、JWT、翻譯器與效能分析設定。 |
| `data` | 建立多資料源用戶端，定義 API、任務、日誌、權限與交易儲存契約。 |
| `errorsx` | 統一建立參數、認證、權限、衝突、限流與內部錯誤。 |
| `job` / `queue` / `sse` / `mcp` | 提供持久化任務、佇列、SSE、MCP 的註冊、執行與生命週期。 |
| `module` / `resource` | 定義模組、模型、遷移、OpenAPI、國際化資源與啟動同步。 |
| `server` | HTTP、gRPC 服務及認證、租戶、校驗、日誌、限流與策略中介軟體。 |
| `api` | 保存公共 protobuf 定義與產生程式碼的獨立 Go 模組。 |
| `client` | 統一建立遠端、服務發現或行程內 gRPC 連線的獨立 Go 模組。 |

## 接入方式

Core 根目錄不維護業務專案的 Wire 組合根。業務專案在自己的 Wire 圖加入 `core.ProviderSet`，並提供：

1. 用於註冊 HTTP、gRPC、MCP 的一個或多個 `module.Module`。
2. 提供模型、遷移、OpenAPI 和國際化檔案系統的 `module.Resource`。
3. 可選的 `job.Tasks`、`queue.Consumers`、`sse.Streams` 集合；不用時提供空集合。
4. `data` 宣告的啟動同步、任務、日誌和權限儲存介面實作。

`NewApp` 會收集依賴並建立 Kratos 應用。HTTP、gRPC、獨立 MCP、獨立 SSE、佇列和 Cron 都是可選服務；缺少對應設定時不會建立監聽器，空集合不會阻止服務啟動。行程內 MCP、SSE 由 HTTP 承載，因此必須同時設定 HTTP。

## 模組契約

目前模組契約只有三個協定註冊方法：

```go
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*http.Server)
	RegisterMCP(*mcp.Server)
}
```

Core 依模組提供順序轉發註冊。模組可只註冊實際使用的協定。不同模組的 OpenAPI 文件、國際化鍵和 SSE 流標識必須唯一，衝突會在組裝時返回錯誤。

## 建置期資源

業務模組實作 `module.Resource`：

| 方法 | 約束 |
| --- | --- |
| `ProjectKey()` / `ProjectName()` | 專案穩定標識和展示名稱；為空時分別回退到 `kratos-core` 和標識。 |
| `Models()` | 按資料源名稱回傳 GORM 模型。模型資料源和預設資料源都必須配置。 |
| `Migrations()` | 回傳包含名稱、檔案系統根路徑和依賴關係的版本化遷移。 |
| `OpenAPI()` | 回傳包含 `openapi.yaml`、`openapi.yml` 或 `openapi.json` 的 `fs.FS`，可附加語言版本。 |
| `I18n()` | 回傳以 locale 命名的 JSON 文案檔案系統；衝突的重複鍵會被拒絕。 |

遷移按依賴拓撲順序執行，資源目錄約定如下：

```text
assets/<version>/<database-type>/README.md
assets/<version>/<database-type>/README.<locale>.md
assets/<version>/<database-type>/<data-source>/README.md
assets/<version>/<database-type>/<data-source>/README.<locale>.md
```

遷移記錄保存檔案引用和 SHA-256 摘要。`resource/migration.Migration.ReadFiles` 會保留檔案順序與邊界，並校驗路徑、資料源、檔案存在性與內容摘要；SQL 摘要使用執行器移除註解後的內容。部署時必須保留歷史遷移檔案，不能將遷移資源目錄直接公開。

## 執行時能力

### 共用上下文與資料

`biz.BaseCase` 聚合啟動上下文、快取、佇列、物件儲存、翻譯器和多資料源資料庫用戶端，並透過 `GetAuthInfo` 取得目前認證主體。`data.Transaction` 統一交易邊界。API、任務、日誌、權限儲存由業務專案實作，Core 只依賴契約。

Core 也提供 `errorsx` 統一錯誤、非同步 `biz.LogPipeline`，以及 `api/common/v1` 的狀態、錯誤、分析圖表、選項、陣列包裝和敏感欄位訊息。

### HTTP 與 gRPC

服務按設定掛載 request ID、國際化、日誌、認證授權、租戶範圍、參數校驗和介面限流中介軟體。HTTP 另外支援：

- `/data/` 本地物件儲存唯讀目錄，可由設定覆蓋根目錄。
- `web` 下的 SPA 子目錄，找不到檔案時回退到 `index.html`。
- `/api/docs/openapi` 和 `/api/docs/swagger` 文件端點。
- 行程內 MCP 和 SSE 路由掛載，路徑可由設定覆蓋。

HTTP、gRPC、MCP 建構時可注入策略解析器。Core 將解析器放入請求上下文，業務包裝器可讀取目前策略；不會重複修改業務回應，並保留取消訊號和 deadline。

### MCP

模組透過 `RegisterMCP` 註冊工具。MCP 支援獨立 HTTP、Legacy SSE、STDIO 和掛載到 Core HTTP 的行程內模式。行程內模式共用 HTTP 監聽器，獨立模式由 Kratos 管理生命週期。

### SSE

`sse.Streams` 註冊業務流，Core 根據 `stream`、`channel`、使用者和租戶解析實際傳輸流。請求會驗證 Bearer 令牌、角色、租戶範圍和快取中的令牌有效性。`sse.SSE` 提供 `Serve`、`PublishJSON`、`PublishEnabled`。HTTP 訂閱會移除服務 deadline，但保留用戶端斷線的取消訊號。

### 佇列

`queue.Server` 管理 Core 日誌、任務日誌和業務消費者。`queue.AddQueue` 投遞 JSON，`ScheduleQueue`、`CancelQueue` 管理延遲訊息，`queue.Decode[T]` 驗證並解碼 `data` 欄位。

### 持久化任務

`job.Tasks` 註冊任務執行器，啟動時重新載入啟用的 `BaseJob`。`job.Job` 提供 `Start`、`Stop`、`Run`，用於排程、移除和立即執行。執行鎖優先使用 Redis；不可用時降級到行程內鎖，只保證單一實例互斥。

### OpenAPI、國際化與同步

OpenAPI 註冊表按專案、語言和介面操作保存文件。重複 key 只有名稱和內容完全相同時才允許，同語言的重複介面會被拒絕；查詢可從請求語言回退到預設文件。

`resource/openapi.OpenAPI` 提供按請求語言、服務名稱、HTTP 路徑和方法查詢文件的執行時入口。

國際化目錄合併 Core 與模組 JSON，內容衝突時組裝失敗。遷移後，啟動同步會將 OpenAPI 轉為介面權限與翻譯快照，同步租戶角色選單，重建資料庫策略並刷新記憶體策略。

## 組裝與啟動順序

1. 解析啟動設定，收集模組資源，按資料源建立資料庫用戶端。
2. 註冊並執行遷移，校驗檔案和依賴順序。
3. 在交易中同步 OpenAPI、國際化、租戶選單和權限；提交後刷新記憶體策略。
4. 建立共用上下文、認證授權、協定服務、佇列、Cron，並註冊模組及獨立集合。
5. 組裝 Kratos App；Wire 產生的清理函式釋放其餘基礎資源。

## 目錄職責

```text
api/                     公共 protobuf 定義和產生程式碼
client/                  gRPC 用戶端、用戶端中介軟體和行程內連線
biz/                     BaseCase、認證授權、日誌流水線和共用執行時
config/                  啟動設定解析
const/                   公共常量
data/                    多資料源用戶端、交易和儲存契約
errorsx/                 統一錯誤建構
job/                     Cron 註冊、持久化任務和生命週期
mcp/                     MCP 服務和生命週期適配
module/                  模組、資源和協定註冊契約
queue/                   佇列投遞、解碼和消費者生命週期
resource/                OpenAPI、國際化、遷移和啟動同步
server/                  HTTP、gRPC、請求 metadata 和中介軟體
sse/                     SSE 流註冊、訂閱和發布
bootstrap.go             ProviderSet 和應用生命週期組裝
Makefile                 生成、格式化、測試、檢查和版本腳本入口
```

## 開發指令

專案要求 Go `1.27.0`。根模組、`api` 和 `client` 是三個獨立 Go 模組，修改任一模組後都應執行對應測試。

```bash
make init                         # 安裝生成、格式化和 Wire 工具
make api                          # 從 proto 生成 api/gen/go
make wire WIRE_DIR=<目錄>         # 在指定組合根生成 Wire 程式碼
make fmt                          # 格式化 Go 原始碼和 import
make test                         # 測試根、api、client 三個模組
make vet                          # 對三個模組執行 go vet
make lint                         # 目前等同於 make vet
make buf-push                     # 生成並推送 protobuf schema（需登入）
make tag                          # 按模組建立已提交程式碼的 patch tag
```

`make tag` 預設處理根模組、`api` 和 `client`：分別使用 `vX.Y.Z`、`api/vX.Y.Z` 和 `client/vX.Y.Z`。可用 `MODULE=api make tag` 指定起始模組，或使用 `EXACT=1` 只處理該模組；腳本不會提交工作區改動。

用戶端連線的獨立說明見 [client/README.md](client/README.md)。
