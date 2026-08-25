# kratos-core

[簡体字中国語](README.md) | [繁体字中国語](README.zh-TW.md) | [English](README.en-US.md) | [日本語](README.ja-JP.md)

`kratos-core` は Kratos サービス向けの共通ランタイムです。ホストプロジェクトは業務 Case、Service、API、プロセスのエントリーポイントを担当し、Core はインフラストラクチャ、トランスポート、リソース登録、アプリケーションのライフサイクルを担当します。

Core は完全な業務テンプレートではありません。ホストは業務サービスとビルド時リソースを `module.Module` として Core に渡し、`NewApp` がまとめて組み立ててホストに渡し、起動します。

## Core の責務

- `bootstrap.Context` からデータベース、Redis、キュー、OSS、JWT、翻訳、プロファイリングの設定を解析します。
- モジュールのリソースに基づいて、複数データベースの GORM クライアント、キャッシュ、キュー、OSS、翻訳器、共有 `biz.BaseCase` を構築します。
- 設定に従って HTTP、gRPC、MCP、SSE、キュー、永続 Cron のランタイムを作成し、ホストサービスを対応するトランスポートへ登録します。
- 組み立て時にデータベースマイグレーション、OpenAPI 同期、テナントのロール・メニュー同期、Casbin ポリシー再構築を実行します。
- オプションのサービスを組み立ててライフサイクルを Kratos に委ね、Wire が生成するクリーンアップ関数で残りの基盤リソースを解放します。

## 公開境界

プロジェクト間で依存できる Go コードは、次の 4 つの入口から提供されます。

- ルートパッケージは `ProviderSet` と `NewApp` を提供します。ホストは通常、自身の Wire グラフに `ProviderSet` を追加するだけです。
- ルートの `biz`、`config`、`const`、`data`、`errorsx`、`job`、`mcp`、`module`、`queue`、`resource`、`server`、`sse` ディレクトリがプロジェクト間の公開パッケージを提供します。
- `api` は独立した Go モジュールです。`api/proto` に Core の protobuf 定義、`api/gen/go` に生成済みの Go 型を保存します。
- `client` は独立した Go モジュールで、`kratos-kit` の設定に基づく gRPC 接続とプロセス内 gRPC 接続を提供します。

`internal/models` 以下のコードは Core の実装詳細であり、プロジェクト間の API ではありません。Core が作成したキャッシュ、キュー、OSS、翻訳器、複数データベースの GORM クライアントは `biz.BaseCase` に注入され、同時に `kratos-kit/sdk.Runtime` にも保存されます。業務コードは必要に応じて BaseCase または SDK から取得できます。

## Wire の統合

ホストは `module.Module` を実装する業務モジュールを提供し、Core で唯一公開されている Wire の入口を使用します。

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

Core の `ProviderSet` は設定、インフラストラクチャ、モジュールリソース、データアクセス、リソース同期、各プロトコルランタイムをまとめ、`NewApp` を含みます。ホスト固有の Provider と `[]module.Module` を追加し、Core の ProviderSet を重複して追加しないでください。Core のルートでは `wire.go` と `wire_gen.go` を管理しません。ホストプロジェクト自身の Wire コマンドで組み立てルートと `wire_gen.go` を生成するか、`make wire WIRE_DIR=<ホストの Wire ディレクトリ>` を使用してください。

## モジュール契約

業務モジュールは `module.Module` を実装します。モジュール自身が業務 Service を保持し、プロトコル登録メソッドで登録します。

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

| メソッド | 役割 |
| --- | --- |
| `RegisterGRPC` | 生成された gRPC Service を登録します。gRPC が設定されていない場合、Core は gRPC Server を作成しません。 |
| `RegisterHTTP` | HTTP Service とルートを登録します。HTTP が設定されていない場合、Core は HTTP Server を作成しません。 |
| `RegisterMCP` | MCP ツールを登録します。MCP は単独で待ち受けることも、HTTP にマウントすることもできます。 |
| `RegisterQueue` | キューコンシューマーを登録します。Core は組み込みのログおよびジョブログコンシューマーも登録します。 |
| `RegisterCron` | 永続データベースタスクの実行器を登録します。通常は `server.RegisterTask` を呼び出します。エラーを返すと組み立てを中止します。 |
| `RegisterSSE` | 業務 SSE ストリームを登録します。通常は `server.RegisterStream` を呼び出します。エラーを返すと組み立てを中止します。 |
| `Resources` | モデル、マイグレーション、OpenAPI、プロジェクトドキュメント、i18n リソースを返します。 |

複数の業務モジュールは、ホストの Wire 組み立てルートから `module.Module` として提供できます。Core は提供順にリソースを収集し、プロトコル登録を転送します。同じドキュメントパス、競合する OpenAPI ドキュメント、内容の異なる i18n メッセージキー、重複する SSE ストリーム ID は組み立て時に拒否されます。

## ビルド時リソース

`module.Resources` は各モジュールが提供する一度きりのリソーススナップショットです。

| フィールド | 内容と制約 |
| --- | --- |
| `ProjectKey` | ドキュメントと OpenAPI の名前空間に使う安定したプロジェクト識別子です。空の場合は `kratos-core` を使用します。 |
| `ProjectName` | プロジェクトの表示名です。空の場合は `ProjectKey` にフォールバックします。 |
| `Models` | データソース名ごとにグループ化された GORM モデルです。モデルを含むすべてのデータソースは設定に存在し、デフォルトデータソースは必須です。 |
| `Migrations` | バージョン管理されたマイグレーションです。各 `module.Migration` は `Name`、`FS`、`Path`、`Dependencies` を宣言し、Core は依存関係の順に実行します。 |
| `OpenAPI` | `openapi.yaml`、`openapi.yml`、`openapi.json` と、任意の `openapi.<locale>.yaml` などの言語ファイルを含む `fs.FS` です。Swagger が有効な場合、Core はプロジェクトと言語ごとに原文と Swagger UI をマウントします。 |
| `Docs` | 通常 `docs.json` を含む `fs.FS` です。ジェネレーターが翻訳を `locale` に書き込み、リクエスト言語で選択して既定本文にフォールバックします。 |
| `I18n` | `zh-CN.json`、`zh-TW.json`、`en-US.json`、`ja-JP.json` などの言語ファイルを含む `fs.FS` です。Core は組み込みメッセージとマージします。 |

ホストは通常、`embed.FS`、コードジェネレーター、`fstest.MapFS` を使って次のリソースを提供します。

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

## ランタイム機能

### 共有コンテキスト

`biz.BaseCase` はホストの業務で共有する基本コンテキストです。`bootstrap.Context`、キャッシュ、キュー、OSS、翻訳器、複数データベースの GORM クライアントを保持し、現在認証されているユーザーを取得する `GetAuthInfo` を提供します。

Core は次の具体的なランタイムサービスもホストへ提供します。

- `job.Job`: データベースの永続タスクを開始、停止、即時実行します。
- `resource/docs.Docs`: マージ済みのプロジェクトドキュメントツリーと、リクエスト言語で選択した本文を検索します。
- `resource/openapi.OpenAPI`: リクエスト言語、Service、HTTP 操作ごとに OpenAPI 情報を検索します。
- `sse.SSE`: SSE サブスクリプションを作成し、JSON イベントを発行します。

### サービスとミドルウェア

HTTP と gRPC のサービスは、設定に応じて request ID、i18n、ログ、認証・認可、バリデーションのミドルウェアを追加します。HTTP はローカル OSS 静的ファイル、SPA フォールバック、Swagger にも対応します。プロセス内 MCP または SSE のエンドポイントは HTTP Server にマウントされるため、いずれかを使用する場合は HTTP の設定が必要です。

キューランタイムは Core のログおよびジョブログメッセージを消費し、ホストが登録したコンシューマーへ転送します。Cron ランタイムはデータベースから有効な `BaseJob` を再読み込みし、モジュールが `RegisterCron` で登録したハンドラーを実行します。

## 組み立てと起動順序

`ProviderSet` と `NewApp` の主な組み立て順序は次のとおりです。

1. 起動設定を解析してモジュールのリソースを収集し、モジュールのモデルからデータソースとマイグレーションレジストリを作成します。
2. データベースマイグレーションを実行し、同じトランザクション内で OpenAPI API、`base_api_i18n` のロケールスナップショット、テナントのロール・メニュー、Casbin のデータベースルールを同期します。ロケール行は `operation + locale` を一意キーに使い、変更される `base_api.id` は参照しません。コミット後にメモリ上のポリシーを更新します。
3. 共有インフラストラクチャ、認証・認可、HTTP/gRPC/MCP/SSE、キュー、Cron のランタイムを作成し、モジュールの登録メソッドを呼び出します。
4. Kratos App を組み立てます。Kratos がトランスポートの起動と停止を管理し、Wire が生成するクリーンアップ関数が残りの基盤リソースを解放します。

## ディレクトリの責務

```text
api/
  proto/common/v1/      Core 共通 protobuf 定義
  gen/go/common/v1/     生成済み protobuf Go コード

client/
  connection.go         リモートまたはプロセス内 gRPC 接続アダプター
  localgrpc/             プロセス内 gRPC サービスの登録と呼び出し

biz/                     共有コンテキスト、認証・認可、公開ランタイムケース
config/                  起動設定の解析
const/                   公開定数
data/                    複数データベースクライアント、トランザクション、Core リポジトリ
errorsx/                 統一エラー生成
job/                     Cron 登録、永続ジョブ、ランタイム
mcp/                     MCP サービスとライフサイクルアダプター
module/                  ホストモジュール、リソース、プロトコル契約
queue/                   キューメッセージヘルパーとコンシューマーのライフサイクル
resource/                ドキュメント、i18n、マイグレーション、OpenAPI、起動同期
  biz/                    API、テナント、Casbin のリソース同期
    dto/                  リソース同期 DTO
  docs/                   プロジェクトドキュメントの登録と検索
    dto/                  プロジェクトドキュメント検索 DTO
  i18n/                   i18n リソースのマージ
  locale/                 ロケール識別子の解析
  migration/              データベースマイグレーション
  openapi/                OpenAPI の登録、検索、HTTP マウント
    dto/                  OpenAPI 検索 DTO
server/                  HTTP、gRPC、ミドルウェア
  middleware/             共通 HTTP/gRPC ミドルウェア
sse/                     SSE ストリームの登録、トランスポート、発行
internal/models/         Core 内部データベースモデル

bootstrap.go             公開 ProviderSet とアプリケーションのライフサイクル組み立て
Makefile                 生成、フォーマット、テスト、静的チェックのコマンド
```

## 開発コマンド

```bash
make tools     # 固定バージョンのコード生成・フォーマットツールをインストール
make api       # api/gen/go を生成
make wire      # WIRE_DIR で指定したホストディレクトリに Wire コードを生成
make fmt       # goimports で Go ソースをフォーマット
make test      # ルート、api、client の 3 つの Go モジュールを検証
make vet       # ルート、api、client の 3 つの Go モジュールを静的検査
make lint      # 現在は make vet と同じ
```

## リリースタグ

`scripts/tag_release.py` は `kratos-kit` に倣い、各 Go module を独立してリリースします。実行前に変更をコミットしてプッシュしてください。スクリプトはリモートのデフォルトブランチ上のコミットだけを確認し、ワークツリーの変更をコミットしません。

```bash
git add -A
git commit -m "コミットメッセージ"
git push origin main

make tag
```

デフォルトではルートモジュール、`api`、`client` をスキャンし、前回のタグ以降にプッシュ済みのコード変更がある場合だけ次の patch タグを作成してプッシュします。

| モジュール | タグ形式 |
| --- | --- |
| ルートモジュール | `vX.Y.Z` |
| `api` | `api/vX.Y.Z` |
| `client` | `client/vX.Y.Z` |

特定のモジュールだけを処理することもできます。

```bash
MODULE=api make tag              # api から再帰的にスキャン
MODULE=api EXACT=1 make tag      # api モジュールだけを処理
```

コード変更がないモジュールやタグが既に存在するモジュールは自動的にスキップします。ルートモジュールの変更検出では `api` と `client` サブモジュールを除外し、サブモジュールの変更でルートタグが重複して作成されないようにします。

プロジェクトは Go `1.27.0` を必要とします。`api` と `client` は独立した Go モジュールのため、どちらかを変更した場合は `cd api && go test ./...` または `cd client && go test ./...` も実行してください。公開モジュールの契約を変更した場合は、Core に依存するホストプロジェクトでも追加のコンパイル確認が必要です。

クライアント接続の個別ガイドは [client/README.md](client/README.md) を参照してください。
