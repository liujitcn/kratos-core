# kratos-core

[简体中文](README.md) | [繁體中文](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` は Kratos サービスの共通ランタイムです。インフラ、プロトコルサーバー、リソース同期、アプリケーションのライフサイクルを組み立てます。業務プロジェクトは Case、Service、API、モデル、プロセスのエントリーポイントを管理します。Core はこのリポジトリの機能だけを提供し、具体的な業務モジュールは含みません。

## 機能

- 起動設定を解析し、複数データベース、キャッシュ、キュー、オブジェクトストレージ、翻訳、認証・認可、プロファイリングを初期化します。
- `module.Module` で HTTP、gRPC、MCP を登録し、`job.Tasks`、`queue.Consumers`、`sse.Streams` でジョブ、キューコンシューマー、SSE ストリームを注入します。
- 設定に応じて HTTP、gRPC、MCP、SSE、キュー、永続 Cron を作成し、Kratos がライフサイクルとエラーを管理します。
- 組み立て時にデータベースマイグレーション、OpenAPI、API ローカライズ、テナントのロール・メニュー、権限ポリシーを同期します。
- 共通コンテキスト、トランザクション境界、ログパイプライン、エラー理由、OpenAPI 検索、SSE 発行、永続ジョブ制御を提供します。
- `api` の共通 protobuf モジュールと、リモート・サービスディスカバリー・プロセス内呼び出しに対応する `client` を提供します。

## 公開パッケージ

| パッケージ | 役割 |
| --- | --- |
| ルートパッケージ | `ProviderSet` と `NewApp` を公開し、設定、リソース、データ、プロトコルをまとめます。 |
| `biz` | `BaseCase`、認証・認可、ログパイプライン、インフラクライアント、共通ランタイム。 |
| `config` | アプリ、データベース、Redis、キュー、ストレージ、JWT、翻訳、プロファイリング設定の解析。 |
| `data` | 複数データベースクライアントと API、ジョブ、ログ、権限、トランザクションの契約。 |
| `errorsx` | 引数、認証、権限、競合、レート制限、内部エラーの統一生成。 |
| `job` / `queue` / `sse` / `mcp` | 永続ジョブ、キュー、SSE、MCP の登録、実行、ライフサイクル。 |
| `module` / `resource` | モジュール、モデル、マイグレーション、OpenAPI、ローカライズ、起動同期。 |
| `server` | HTTP、gRPC と認証、テナント、検証、ログ、レート制限、ポリシー middleware。 |
| `api` | 共通 protobuf 定義と生成コードを持つ独立 Go モジュール。 |
| `client` | リモート、ディスカバリー、プロセス内 gRPC 接続を作る独立 Go モジュール。 |

## 組み込み

Core のルートには業務プロジェクトの Wire 合成ルートを置きません。業務プロジェクト自身の Wire グラフに `core.ProviderSet` を追加し、次を提供します。

1. HTTP、gRPC、MCP を登録する `module.Module`。
2. モデル、マイグレーション、OpenAPI、ローカライズファイルを返す `module.Resource`。
3. 必要に応じた `job.Tasks`、`queue.Consumers`、`sse.Streams`。使わない機能は空の集合にします。
4. 起動同期、ジョブ、ログ、権限に使う `data` のストレージ契約の実装。

`NewApp` は依存関係を集めて Kratos アプリを作成します。HTTP、gRPC、独立 MCP、独立 SSE、キュー、Cron は任意です。対応する設定がなければ listener は作られず、空の集合でもサービス起動を妨げません。プロセス内 MCP と SSE は HTTP にマウントするため、HTTP 設定も必要です。

## モジュール契約

現在の契約は次の 3 つの登録メソッドです。

```go
type Module interface {
	RegisterGRPC(grpc.ServiceRegistrar)
	RegisterHTTP(*http.Server)
	RegisterMCP(*mcp.Server)
}
```

Core はモジュールの順番で登録を転送します。OpenAPI、ローカライズキー、SSE ストリーム ID は全モジュールで一意でなければならず、競合時は組み立てを失敗させます。

## ビルド時リソース

`module.Resource` の契約は次のとおりです。

| メソッド | 契約 |
| --- | --- |
| `ProjectKey()` / `ProjectName()` | 安定した識別子と表示名。空の場合は `kratos-core` と識別子にフォールバック。 |
| `Models()` | データソース名ごとの GORM モデル。モデルのデータソースと既定データソースは設定必須。 |
| `Migrations()` | 名前、ファイルシステム、依存関係を持つバージョン付きマイグレーション。 |
| `OpenAPI()` | `openapi.yaml`、`openapi.yml`、`openapi.json` と任意の locale ファイルを含む `fs.FS`。 |
| `I18n()` | locale 名の JSON ファイル。競合する重複キーは拒否。 |

マイグレーションは依存順に実行し、次の配置を使います。

```text
assets/<version>/<database-type>/README.md
assets/<version>/<database-type>/README.<locale>.md
assets/<version>/<database-type>/<data-source>/README.md
assets/<version>/<database-type>/<data-source>/README.<locale>.md
```

記録にはファイル参照と SHA-256 が保存されます。`resource/migration.Migration.ReadFiles` は順序と境界を保ったまま、パス、データソース、存在、内容を検証します。SQL の摘要は実行時にコメントを除いた内容から計算されます。過去のマイグレーションファイルは保持し、公開静的ファイルとして配信しないでください。

## ランタイム機能

### 共通コンテキストとデータ

`biz.BaseCase` は起動コンテキスト、キャッシュ、キュー、オブジェクトストレージ、翻訳器、複数データベースクライアントを保持し、`GetAuthInfo` で認証主体を取得できます。`data.Transaction` はトランザクション境界を定義します。API、ジョブ、ログ、権限ストアは業務プロジェクトが実装し、Core は契約だけに依存します。

`errorsx` の統一エラー、非同期 `biz.LogPipeline`、`api/common/v1` の状態・エラー・分析・オプション・配列・Secret メッセージも提供します。

### HTTP と gRPC

設定に応じて request ID、ローカライズ、ログ、認証・認可、テナント範囲、検証、レート制限 middleware を追加します。HTTP は `/data/` のローカルストレージ、`web` の SPA（`index.html` フォールバック）、`/api/docs/openapi` と `/api/docs/swagger` のドキュメント、プロセス内 MCP/SSE のマウントをサポートします。

HTTP、gRPC、MCP はポリシーリゾルバーを受け取り、リクエストコンテキストへ伝播します。Core はレスポンスを二重に変更せず、キャンセルと deadline を保持します。

### MCP

モジュールは `RegisterMCP` でツールを登録します。独立 HTTP、Legacy SSE、STDIO、HTTP へのプロセス内マウントに対応します。プロセス内は HTTP listener を共有し、独立モードは別ライフサイクルで動作します。

### SSE

`sse.Streams` は業務ストリームを宣言します。`stream`、`channel`、ユーザー、テナントから実際のストリームを解決し、Bearer token、ロール、テナント範囲、キャッシュ上の token 有効性を検証します。`sse.SSE` は `Serve`、`PublishJSON`、`PublishEnabled` を提供します。HTTP 購読ではサービス deadline を外し、クライアント切断のキャンセルは保持します。

### キュー

`queue.Server` は Core ログ、ジョブログ、業務コンシューマーを処理します。`queue.AddQueue` で JSON を送り、`ScheduleQueue` と `CancelQueue` で遅延メッセージを管理し、`queue.Decode[T]` で `data` フィールドを検証・デコードします。

### 永続ジョブ

`job.Tasks` が実行器を登録し、起動時に有効な `BaseJob` を再ロードします。`job.Job` の `Start`、`Stop`、`Run` でスケジュール、削除、即時実行を行います。実行ロックは Redis を優先し、利用できない場合は単一プロセス内ロックへフォールバックします。

### OpenAPI、ローカライズ、同期

OpenAPI レジストリはプロジェクト、言語、操作ごとに文書を保持します。同一 key は名前と内容が同じ場合だけ重複を許可し、同一言語の重複操作は拒否します。言語が不足する場合は既定文書へフォールバックできます。

`resource/openapi.OpenAPI` はリクエスト言語、サービス名、HTTP パス、メソッドで文書を検索するランタイム入口です。

ローカライズカタログは Core とモジュールの JSON を統合し、競合キーで失敗します。マイグレーション後に OpenAPI から権限・翻訳スナップショットを作り、テナントロールのメニューとデータベースポリシーを同期してメモリ上のポリシーを更新します。

## 組み立て順序

1. 起動設定、モジュールリソース、データソース別クライアントを準備します。
2. マイグレーションを登録・実行し、ファイルと依存関係を検証します。
3. OpenAPI、ローカライズ、テナントメニュー、ポリシーをトランザクションで同期し、commit 後にポリシーを更新します。
4. 共通コンテキスト、認証・認可、プロトコル、キュー、Cron を作り、モジュールと集合を登録します。
5. Kratos App を組み立てます。Wire の cleanup が残りのリソースを解放します。

## ディレクトリ

```text
api/                     共通 protobuf と生成コード
client/                  gRPC クライアント、middleware、プロセス内接続
biz/                     BaseCase、認証、ログパイプライン、共通ランタイム
config/                  起動設定の解析
const/                   公開定数
data/                    複数 DB、トランザクション、ストレージ契約
errorsx/                 統一エラー
job/                     Cron、永続ジョブ、ライフサイクル
mcp/                     MCP サービスとライフサイクル
module/                  モジュール、リソース、プロトコル契約
queue/                   キュー送信、デコード、コンシューマー
resource/                OpenAPI、ローカライズ、マイグレーション、同期
server/                  HTTP、gRPC、request metadata、middleware
sse/                     SSE 登録、購読、発行
bootstrap.go             ProviderSet とアプリ組み立て
Makefile                 生成、整形、テスト、検査、バージョン操作
```

## 開発コマンド

Go `1.27.0` が必要です。ルート、`api`、`client` は独立した Go モジュールです。

```bash
make init                         # 生成、整形、Wire ツールを導入
make api                          # proto から api/gen/go を生成
make wire WIRE_DIR=<directory>    # 指定した Wire ルートで生成
make fmt                          # Go ソースと import を整形
make test                         # 3 モジュールをテスト
make vet                          # 3 モジュールで go vet
make lint                         # make vet の別名
make buf-push                     # protobuf schema を push（ログインが必要）
make tag                          # commit 済み変更の patch tag を作成
```

`make tag` は `vX.Y.Z`、`api/vX.Y.Z`、`client/vX.Y.Z` を作成します。`MODULE=api make tag` で開始モジュールを指定し、`EXACT=1` でそのモジュールだけを処理できます。作業ツリーを commit することはありません。

クライアント接続の詳細は [client/README.md](client/README.md) を参照してください。
