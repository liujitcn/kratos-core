# kratos-core

[簡體中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md)

`kratos-core` 是 Kratos 服務的基礎執行時。宿主負責業務服務和程序入口，Core 負責設定、基礎元件、HTTP/gRPC/MCP/SSE、資源同步以及統一生命週期。

## 公開邊界

Core 的公開 Go 程式碼保持為三個層次：

- 根套件僅用 `wire.go` 提供宿主 Wire 圖需要的根 Provider、SDK Runtime 和應用建構函式。
- `pkg` 僅保留 `assets`、`biz`、`const`、`errorsx`、`module` 五個公開套件。
- `api/gen/go` 提供跨專案共用的 protobuf 類型。

其他實作全部位於 `internal`，不屬於跨專案 API。快取、佇列、預設資料庫、OSS 和翻譯器由 Core 建立後寫入 `kratos-kit/sdk.Runtime`，業務程式碼直接從 SDK 取得，不再經過 Core 二次封裝。

## Wire 接入

宿主在自己的 Wire 組合根中使用 `ProviderSet` 初始化 Core 根物件並逐項注入業務能力，同時使用 `biz.ProviderSet` 建立共用的 `BaseCase`：

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

`newHostModuleResources` 回傳 `module.Resources`。Core 先收集模型、遷移、OpenAPI、專案文件和 I18n，初始化基礎設施並寫入 `sdk.Runtime`；Wire 隨後建立宿主 Case 和 Service，最後把完整 `module.Module` 交給 Core 啟動。

## 模組契約

業務模組實作 `pkg/module.Module`，服務註冊仍由模組自己負責：

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

`module.Resources` 是業務建立前的靜態貢獻，包含模型、遷移、OpenAPI、專案文件和 I18n；`module.Contributions` 只包含任務、SSE 和 AI 註冊函式。HTTP、gRPC、MCP、定時任務等服務以及中介軟體均由 Core 統一建立和管理。宿主透過 `Contributions.AI` 接收 `module.AIRegistrar` 並註冊工具與固定流程。

`pkg/module` 描述宿主向 Core 提供的內容，並集中保存 Admin 實際使用的 AI 註冊與執行契約；這些共用型別直接定義在 `pkg/module`，Core `internal/agent` 只實作或使用它們，不允許 `pkg/module` 反向引用 `internal`。`pkg/biz.AI` 和 `pkg/biz.Docs` 一樣只提供唯讀查詢，其中 AI 僅公開 `Tools` 與 `FixedFlowProviders`；`biz.ProviderSet` 只建立 `BaseCase`。

## 建置期資源

宿主透過 `module.Resources` 直接提供建構期資源：

| 欄位 | 用途 |
| --- | --- |
| `OpenAPI` | OpenAPI YAML/JSON 檔案系統。 |
| `Docs` | 專案文件產生結果，通常包含 `docs.json`。 |
| `I18n` | 宿主語言 JSON 檔案系統。 |

資料庫遷移透過 `module.Resources.Migrations` 提供，每個 `module.Migration` 明確宣告模組名稱、檔案系統、版本目錄根路徑和依賴模組。

宿主只為 OpenAPI、Docs 和 I18n 提供 `fs.FS`。Core 讀取後轉換為內部註冊表和 `pkg/biz` 查詢所需的資料結構；國際化目錄實作位於 `internal/i18n`，語言狀態和請求上下文實作位於 `internal/locale`。

## 目錄職責

```text
api/
  proto/                 Core 公共 protobuf 定義
  gen/go/                protobuf 產生的 Go 程式碼

pkg/
  assets/                宿主建置期資源和文件 DTO
  biz/                   Core 向宿主提供的業務能力及 BaseCase Wire Provider
  const/                 公共常數
  errorsx/               統一錯誤建構
  module/                宿主向 Core 提供的資源、協議註冊、AI 與執行期貢獻契約

internal/
  application/           應用組裝、模組資源收集和統一生命週期
  runtime/               cache、queue、database、auth、OSS 和 AI 初始化
  data/                  Core 最小模型和儲存庫
  server/                HTTP、gRPC、MCP、SSE 服務組裝
  agent/, task/, job/    AI、任務和 Cron 內部實作
  docs/, i18n/, locale/, openapi/ 文件、國際化、語言狀態和 OpenAPI 內部實作

wire.go                  對宿主公開的 Root、Runtime 和應用 Wire 門面
```

## 開發指令

```bash
make api
make fmt
make test
make vet
make lint
```

專案要求 Go `1.26.5`。修改公開契約後需要同時驗證 `kratos-admin` 和 `kratos-shop` 的編譯。
