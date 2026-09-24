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

詳細・設計判断の理由はすべて `docs/architecture.md` が正本。要点のみ:

**Feature 単位のシンプルな3層**。Clean Architecture / DDD は採用するものではなく、実装上の痛みが出た箇所に必要な分だけ導入する設計手法（学習用なので痛みは意図的に作る）。

```txt
handler → service → repository → DB(生SQL)
(net/http)
```

### ディレクトリ構成（予定）

- `cmd/server/main.go` — 起動・DB接続・ServeMux配線（依存注入はここ）
- `internal/todo/` — Feature 単位で1パッケージ
  - `handler.go` — HTTP受付。`http.ResponseWriter`/`*http.Request` を触るのはここだけ
  - `service.go` — ユースケース・業務処理
  - `repository.go` — 生SQL実装（`database/sql` + SQLite）
  - `todo.go` — Todo構造体
- `api/*.http` — 動作確認用リクエスト。Phase 1 はこれを主体にコードを書いて体感する
- `api/openapi.yaml` — 外部契約。Phase 1 では後追いでよく、Phase 2 以降で正本にする

### 守るべき制約

- handler → repository の直接呼び出し禁止。handlerにSQLを書かない
- serviceに `*http.Request`/`http.ResponseWriter` を渡さない
- repositoryにHTTP概念を持ち込まない
- `api/openapi.yaml` を正本にした後は、実装は契約に従う（契約を後追いで勝手に変えない）
- フレームワーク（gin等）・ORMは使わない。`net/http` 標準（Go 1.22+ の `http.ServeMux` メソッド別ルーティング・`r.PathValue`）と生SQL
- **先回りして抽象化しない**: Phase 1 では interface / Value Object / Aggregate を作らない。導入タイミングは docs/architecture.md の「学習フェーズ」参照

## ドキュメント

- Markdownは `.markdownlint.jsonc` の設定に従う（行長制限なし、コードブロック内タブ許容）
