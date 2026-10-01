# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 最重要：AIの役割（README.md より）

このリポジトリは **GoでTodo APIをフルスクラッチ手書き実装する学習用リポジトリ**。ユーザーが自分の手でコードを書くこと自体が目的。

- **コードを必要以上に実装しない。** ユーザーからの質問への回答・解説・レビューに徹する。
- コードライティング以外の補助（コマンド実行、動作確認、curl作成など）は制限なく行ってよい。
- OpenAPI（`api/openapi.yaml`）については例外的に、手書きされた契約の構文・網羅性・整合性の補正をAIが担う。ただ直すのではなく、何がまずいのかフィードバックすることが一番の目的。

## コマンド

標準のGoツールチェーンのみ（Makefile等なし）。モジュール名: `github.com/AugusTaro/my-awesome-practice-go`（Go 1.23）。

```sh
go build ./...                    # ビルド
go test ./...                     # 全テスト
go test ./internal/todo/ -run TestXxx   # 単一テスト
go vet ./...                      # 静的解析
go run ./cmd/server               # サーバ起動
```

## 設計の指針

設計の考え方は `docs/architecture.md` を参照。レビュー時は以下を判断軸にし、ルール違反の指摘ではなく「今しんどさが出ているか」で助言する。

- **後から変えると高いものだけ最初に固定する**: HTTP 境界（`internal/adapter/web`）、永続化の境界（Feature ごとの Store）、依存の向き（`adapter → feature`、コアは外界を知らない）
- **層・interface・型は、そうしないとしんどいと感じた瞬間に足す**。先回りして抽象化しない
- **責務は所有する場所に置く**: HTTP の入出力は adapter、永続化の詳細は Store、ドメインのルールはそれを持つ型の近く
- `net/http` 標準のみ。フレームワーク・ORM は使わず、DB は生SQL

## ドキュメント

- Markdownは `.markdownlint.jsonc` の設定に従う（行長制限なし、コードブロック内タブ許容）
