# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 最重要：AIの役割（README.md より）

このリポジトリは **GoでTodo APIをフルスクラッチ手書き実装する学習用リポジトリ**。ユーザーが自分の手でコードを書くこと自体が目的。

- **コードを必要以上に実装しない。** ユーザーからの質問への回答・解説・レビューに徹する。
- コードライティング以外の補助（コマンド実行、動作確認、curl作成など）は制限なく行ってよい。
- OpenAPI（`api/openapi.yaml`）については例外的に、手書きされた契約の構文・網羅性・整合性の補正をAIが担う方針（docs/architecture.md 参照）ただ直すのではなく、何がまずいのかフィードバックすることが一番の目的。

## コマンド

標準のGoツールチェーンのみ（Makefile等なし）。モジュール名: `github.com/AugusTaro/my-awesome-practice-go`（Go 1.23）。

```sh
go build ./...                    # ビルド
go test ./...                     # 全テスト
go test ./internal/service/ -run TestXxx   # 単一テスト
go vet ./...                      # 静的解析
go run ./cmd/server               # サーバ起動（実装後）
```

## アーキテクチャ

詳細・コード例・設計判断の理由はすべて `docs/architecture.md` が正本。要点のみ:

**軽量クリーンアーキテクチャ + DDD-lite**（契約駆動）

```txt
OpenAPI(契約を先に確定) → handler → service → repository(interface) → DB(生SQL)
                          (net/http)          ※interfaceは利用側=serviceが定義
```

### ディレクトリ構成（予定）

- `cmd/server/main.go` — 起動・DB接続・ServeMux配線（依存注入はここ）
- `internal/handler/` — HTTP受付。`http.ResponseWriter`/`*http.Request` を触るのはここだけ
- `internal/service/` — ビジネスロジック+段取り。**repository interfaceをここに定義**（consumer-defined interface）
- `internal/repository/` — 生SQL実装（`database/sql` + SQLite）。interfaceをimportせず暗黙的に満たす
- `internal/model/` — 構造体+振る舞いメソッド（軽量entity。ビジネスルールはここに置く）
- `internal/dto/` — HTTP入出力用構造体
- `api/openapi.yaml` — 外部契約の正本。実装より先に書く。`api/*.http` で動作確認

### 守るべき制約

- handler → repository の直接呼び出し禁止。handlerにSQLを書かない
- serviceに `*http.Request`/`http.ResponseWriter` を渡さない
- repositoryにHTTP概念を持ち込まない
- modelを直接HTTPレスポンスにしない（dto経由）
- 実装はOpenAPIの契約に従う（契約を後追いで勝手に変えない）
- フレームワーク（gin等）・ORMは使わない。`net/http` 標準（Go 1.22+ の `http.ServeMux` メソッド別ルーティング・`r.PathValue`）と生SQL
- **DDDをやりすぎない**: 1周目はValue Object/Aggregateを先回りして作らない。2周目でルールが生まれた時に導入（判断軸は docs/architecture.md の2周目セクション参照）

## ドキュメント

- Markdownは `.markdownlint.jsonc` の設定に従う（行長制限なし、コードブロック内タブ許容）
