# Go Todoアプリ アーキテクチャ学習ガイド（決定版）

## 〜 軽量クリーンアーキテクチャ + DDD-lite 〜

## このドキュメントの位置づけ

AIを使った開発で「コードの流れは追えるが、細部まで腑に落ちていない」状態を解消するため、
モダンで無難なGoのWebバックエンド構成を**フルスクラッチ・手書き**で一本やり切るための指針。

ゴールは設計の「正解」を暗記することではなく、

> **なぜその構造にするのかを自分の言葉で説明でき、状況に応じて設計を選べるようになること**

最終的に、現在の業務（マイクロサービス / クリーンアーキ寄りレイヤード / DDD）のコードを
高い解像度で読み替えられる状態を目指す。

---

## 採用するアーキテクチャ：軽量クリーンアーキテクチャ + DDD-lite

「デファクト版」と呼ぶ。2025〜2026年のGo Webバックエンドで最も無難でスタンダードな構成。
教科書的クリーンアーキテクチャ／フルDDDから「実利のある部分だけ」を採り、Go流に削ったもの。

```txt
OpenAPI(契約) ── 先に確定 ──┐
                            ↓ 満たすように実装
handler → service → repository(interface) → DB(生SQL)
(net/http)          （interfaceは利用側=service が定義）
```

### この構成の設計判断（重要）

- **interfaceは入れる、ただし利用側が定義する**（Goイディオム / 実利）
  repositoryのinterfaceを **service側に定義**し（consumer-defined interface）、実装はrepositoryパッケージに置く。serviceのテストでDBを偽物に差し替えられる。「具象でなく抽象に依存する」(DIP) は守る。
- **層の細分化はしない**（教条を避ける）
  usecase層とdomain層を分けない。`service`が両方を兼ねる。
- **modelは1つ・振る舞いを持たせる**（DDD-liteの入り口）
  entity用とDB用に構造体を分けない（詰め替えコストを払わない）。ただしstructにメソッドを持たせ、ビジネスルールをそこに置く（貧血モデルを避ける）。
- **ORMを使わず生SQLを書く**（学習目的）
  `database/sql` 標準ライブラリで手書きし、repositoryが実際に何をしているかを完全に自分の手を通す。
- **フレームワークを使わず net/http 標準で書く**（依存最小 / Go 1.22+）
  Go 1.22 で `http.ServeMux` がメソッド別ディスパッチとパスパラメータ（`GET /todos/{id}`）に対応し、ルーティングのためだけにginを入れる動機が大きく減った。handlerも標準ライブラリで書き、フレームワークが隠す「リクエストのパース・レスポンス書き込み・ステータス設定」を自分の手を通す。生SQLと同じ発想。
- **OpenAPIを先に書く（契約駆動 / スキーマ・ファースト）**
  実装より先に OpenAPI で外部契約を確定し、それを満たすように実装する。フロントのGUIは作らず、契約を正本に curl / `.http` で叩く。手書きで契約設計の思考を通し、生成ツール・AIで構文と網羅性を補正する折衷をとる。

### なぜ「厳密なクリーンアーキ / フルDDD」を採用しないか

クリーンアーキの「考え方」（ビジネスロジックをHTTP/DBから切り離す）は今も有効。
否定されているのは**教条的な全部入り適用**。特にGoでは行き過ぎた抽象化への揺り戻しが起きている。

- 実装が1つしかないrepositoryに律儀にinterfaceを切るのはテスト目的以外ではオーバーエンジニアリング気味。
- 層ごとにstructを詰め替える変換コード（entity ⇄ model ⇄ DTO）が増え、ロジックより変換が多くなる「マッパー地獄」。
- 全層で依存性逆転を徹底すると、追うべきファイルが増えてむしろ読みにくい。

→ Goの文化（必要になるまで抽象化しない）に合わせ、**interfaceだけ残し層は薄く保つ**のが今の無難解。

### Todoという題材についての注意（重要）

DDDが本領を発揮するのは「業務ルールが豊かなドメイン」。Todoは本来ほとんどルールがなく、
フルにDDDしようとすると「ルールが無いのに無理やりオブジェクトを作る」過剰設計を自分で再現してしまう。

そこで本ガイドは二段構えにする。

- **1周目** … 軽量クリーンアーキの骨格づくり。DDDは最小限（modelに振る舞いを持たせる程度）。Value ObjectやAggregateは先回りして作らない。
- **2周目** … ドメインに**あえてルールが生まれる拡張**を入れ、DDD-liteを効かせる（不変条件・Value Object・Aggregateの入り口）。

> 1周目で「箱と型」を作り、2周目で「ルールという中身」を流し込んでDDDに血を通わせる。

---

## フロント / API契約の方針（契約駆動）

ブラウザのGUI（React等）は**作らない**。この学習の目的はバックエンドのアーキテクチャ理解であり、
フロントを作り込むとフォーカスが分散するため。フルスタックフレームワークが意図的に曖昧にする
「HTTP・層・依存方向の境界」こそ、ここで鍛えたい対象。業務のマイクロサービス（フロントを持たずAPIを公開する側）とも地続き。

### 進め方：OpenAPIファースト（スキーマ・ファースト）

実装より先に **OpenAPI で外部契約を確定**し、その契約を満たすように実装する。
OpenAPI＝外部契約のスキーマ設計であり、コードの実装より重い意思決定。先に固めるのが筋。

```txt
1. api/openapi.yaml を書く（パス・メソッド・リクエスト/レスポンススキーマ・エラー・required）
2. 契約をレビューして確定（これが正本 = source of truth）
3. 契約を満たすように handler / dto / service / repository を実装
4. .http / curl で契約通りか検証（AIにcurlを組ませてもよい）
```

### 手書き vs 生成のバランス（現実解）

- **手書き**：契約設計の「なぜこの形か」を考える思考を通すため、骨格は手で書く。
- **生成 / AI補正**：OpenAPIは仕様が細かく（スキーマ参照、エラー定義、`required`漏れ等）、フル手書きは抜けが出る。構文・網羅性・整合性はツールやAIで補正する。
- 結論：**手で設計し、ツール/AIで精度を担保する**折衷。1周目は手書き比率を高め、慣れたら `oapi-codegen` 等のコード生成も検討。

### 動作確認：.http / curl

- `api/*.http`（VS Code REST Client 等）に手書きのリクエストを溜める＝叩ける仕様書になる。
- curlはぱっと書けるよう手書きの練習も兼ねる。OpenAPIを正本、`.http`/curlを手元の動作確認、という二層構成。

### コード生成の位置づけ（将来）

- `oapi-codegen`：OpenAPIから handler のインターフェースや型を生成（契約を常にソース・オブ・トゥルースに保つ）。
- ただし生成ツールは「素を学ぶ」目的を一部隠すので、1周目は手実装で素を通し、2周目以降に導入を検討する。

---

## ディレクトリ構成

```txt
cmd/
  server/
    main.go
internal/
  handler/          # HTTP受付。http.ResponseWriter / *http.Request を触るのはここだけ
  service/          # ビジネスロジック + 操作の段取り。repository interface もここに定義
  repository/       # repository実装（生SQL）
  model/            # データ構造 + 振る舞いメソッド（= 軽量entity）
  dto/              # HTTP入出力用の構造体
api/
  openapi.yaml      # 外部契約（正本）。実装より先に書く
  *.http            # 動作確認用のリクエスト（手書きcurl相当）
```

---

## レイヤー責務

### handler（インターフェース層）

- HTTPレイヤ。リクエスト受け取り → DTOへ変換 → service呼び出し → レスポンス返却
- `http.ResponseWriter` / `*http.Request` を触ってよいのはここだけ
- OpenAPIで定義した契約（パス・ステータス・レスポンス形）を満たす責任を持つ
- ❌ ビジネスロジックを書かない / ❌ SQLを書かない

### service（アプリケーション + ビジネスロジック層）

- バリデーション、ビジネスルールの呼び出し、処理の流れの制御、トランザクション境界
- **repositoryのinterfaceをこのパッケージに定義し**、それ越しに呼ぶ（consumer-defined interface）
- ❌ HTTP（`http.Request` 等）を持たない / ❌ SQLを直接書かない

### repository（データアクセス層）

- service側が定義したinterfaceを満たすよう、生SQLで実装する
- `database/sql` を使い、プレースホルダ・Scan・Rowsクローズ・トランザクションを自分で扱う
- ❌ HTTPの概念を持たない

### model（軽量entity）

- データ構造を表す。**振る舞い（メソッド）を持つ**
- 例:「完了済みなら再完了できない」というルールは `todo.Complete()` に置く（貧血モデルを避ける）

### dto（HTTP入出力）

- リクエスト/レスポンスのための構造体。modelとは別に持つ

---

## 依存関係ルール

```txt
handler → service → repository(interface ← serviceが所有)
```

### 禁止事項

- handler → repository（直接呼ばない）
- handlerにSQLを書く
- serviceに `*http.Request` / `http.ResponseWriter` を渡す
- repositoryにHTTPの概念を持ち込む
- modelをDTOとして直接HTTPで返す（dtoを経由する）

---

## データの流れ

```txt
HTTP Request
↓
DTO（入力）
↓
handler
↓
service
↓
model（振る舞い・ルール）
↓
repository(interface)
↓
repository実装（生SQL）
↓
DB
↓
model
↓
service
↓
DTO（出力）
↓
HTTP Response
```

---

## 各層の最小コード例

### model（振る舞いを持つ = 軽量entity）

```go
package model

import (
	"errors"
	"time"
)

type Todo struct {
	ID        int
	Title     string
	Completed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ビジネスルールを model のメソッドに置く（貧血モデルを避ける）
func (t *Todo) Complete() error {
	if t.Completed {
		return errors.New("todo is already completed")
	}
	t.Completed = true
	t.UpdatedAt = time.Now()
	return nil
}
```

### service（interfaceを"利用側"で定義し、生SQL実装を差し込む）

```go
package service

import (
	"context"
	"errors"

	"example/internal/model"
)

// repository interface は利用側（service）が、必要なメソッドだけ宣言する
// （consumer-defined interface = Goイディオム）
// repository実装はこのinterfaceの存在を知らなくてよい。
type todoRepository interface {
	FindByID(ctx context.Context, id int) (*model.Todo, error)
	Update(ctx context.Context, t *model.Todo) error
}

type TodoService struct {
	repo todoRepository // 具象ではなく抽象(interface)に依存する
}

func NewTodoService(repo todoRepository) *TodoService {
	return &TodoService{repo: repo}
}

// 「Todoを完了する」という1操作の段取り
func (s *TodoService) CompleteTodo(ctx context.Context, id int) (*model.Todo, error) {
	todo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if todo == nil {
		return nil, errors.New("todo not found")
	}

	// ビジネスルールは model 側に委ねる
	if err := todo.Complete(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}
```

### repository（interfaceの存在を知らずに、ただ実装する / 生SQL）

```go
package repository

import (
	"context"
	"database/sql"

	"example/internal/model"
)

// service側の todoRepository interface を、暗黙的に満たす。
// このパッケージは interface を import すらしない。
type TodoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) FindByID(ctx context.Context, id int) (*model.Todo, error) {
	query := `SELECT id, title, completed, created_at, updated_at
	          FROM todos WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var t model.Todo
	if err := row.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // 見つからない時の扱いはチームで決める（ErrNotFoundを返す流儀もある）
		}
		return nil, err
	}
	return &t, nil
}

func (r *TodoRepository) Update(ctx context.Context, t *model.Todo) error {
	query := `UPDATE todos SET title = ?, completed = ?, updated_at = ?
	          WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, t.Title, t.Completed, t.UpdatedAt, t.ID)
	return err
}
```

### handler（net/http標準。ロジックゼロ）

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"example/internal/service"
)

type TodoHandler struct {
	svc *service.TodoService
}

func NewTodoHandler(svc *service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// 小さなヘルパ（JSONレスポンスは自分で書く＝フレームワークが隠していた部分）
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// シグネチャは標準の http.HandlerFunc。gin.Context は登場しない。
func (h *TodoHandler) Complete(w http.ResponseWriter, r *http.Request) {
	// Go 1.22+: パスパラメータは r.PathValue で取れる
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}

	todo, err := h.svc.CompleteTodo(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// 本来は dto に詰め替えて返す（OpenAPIのレスポンススキーマに一致させる）
	writeJSON(w, http.StatusOK, todo)
}
```

### 配線（main.go：依存を上から注入し、標準ServeMuxに登録）

```go
// db *sql.DB は接続済みとする
repo := repository.NewTodoRepository(db)   // *repository.TodoRepository
svc := service.NewTodoService(repo)        // repo は todoRepository interface を満たす
h := handler.NewTodoHandler(svc)

mux := http.NewServeMux()
// Go 1.22+: "METHOD /path/{param}" 形式が標準で書ける（ginのルーター不要）
mux.HandleFunc("PATCH /todos/{id}/complete", h.Complete)
// mux.HandleFunc("POST /todos", h.Create) など

http.ListenAndServe(":8080", mux)
```

---

## 1周目：シンプルCRUD（軽量クリーンアーキの骨格 + 最小DDD）

### 目的

- レイヤー分割を体で覚える
- 生SQLでrepositoryの責務を理解する
- interfaceを利用側で定義する意味を手で書いて体感する
- modelに振る舞いを持たせる（最小DDD）

### 技術スタック

- Go（1.22+ / 強化された `http.ServeMux` を使う）
- `net/http` 標準ライブラリ（HTTPルーティング・handler）
- SQLite（`database/sql` + ドライバ）
- OpenAPI（外部契約の正本）+ `.http` / curl（動作確認）
- ❌ Webフレームワーク（gin等）は使わない（標準で書く）
- ❌ ORMは使わない（生SQLを書く）

### 機能

- Todo作成 / 一覧取得 / 詳細取得 / 更新 / 削除

### API

```txt
POST   /todos
GET    /todos
GET    /todos/:id
PATCH  /todos/:id
DELETE /todos/:id
```

### テーブル

```txt
id         INTEGER PRIMARY KEY AUTOINCREMENT
title      TEXT NOT NULL
completed  BOOLEAN NOT NULL DEFAULT false
created_at DATETIME NOT NULL
updated_at DATETIME NOT NULL
```

### 実装ステップ（契約駆動：OpenAPIを先に書く）

1. **`api/openapi.yaml` を書く**（外部契約を先に確定。手書きで設計し、ツール/AIで構文・網羅性を補正）
2. `main.go`（起動・DB接続・ServeMux配線）
3. model作成（構造体 + 振る舞いメソッド）
4. service作成（repository interfaceの定義 + 段取り）
5. repository作成（interfaceを満たす生SQL実装）
6. dto作成（OpenAPIのスキーマに対応する入出力構造体）
7. handler作成（HTTPのみ。契約に一致するレスポンスを返す）
8. `.http` / curl で動作確認（OpenAPIの契約通りか検証）

### 制約（重要）

- handlerにSQLを書かない
- serviceに `*http.Request` / `http.ResponseWriter` を渡さない
- repositoryにHTTP概念を持ち込まない
- ビジネスルールはmodelのメソッドに置く
- 実装はOpenAPIの契約に従う（契約を後追いで勝手に変えない）
- **DDDをやりすぎない**：ルールが無いのにValue Object/Aggregateを先回りして作らない

### 学習ポイント

- **契約駆動**：OpenAPIを先に書き、契約を正本に実装する感覚
- **依存関係**：なぜhandlerからrepositoryを直接呼ばないのか
- **責務分離**：なぜhandlerにロジックを書かないのか
- **net/http標準**：フレームワークが隠していたパース・レスポンス書き込み・ルーティングを自分で扱う
- **interface（利用側定義）**：なぜserviceが必要なメソッドだけ宣言するのか／テストで実感する
- **DIP**：serviceが具象でなく抽象に依存するとは何か
- **生SQL**：プレースホルダ・Scan・Rowsクローズを自分で扱う
- **最小DDD**：ルールをmodelのメソッドに置く感覚

---

## 2周目：拡張フェーズ（ルールを生んでDDD-liteを効かせる）

### 目的

- ドメインに**あえてルールが生まれる拡張**を入れ、DDDの戦術パターンを"必要だから使う"形で体感する
- 「いつ層・道具を増やすか」の判断軸を作る

> ポイント：パターンを機械的に全部やるのではなく、**ルールが生まれた瞬間に対応する道具として導入する**。

### 拡張案1：期限・優先度（不変条件 + Value Object）

- 追加カラム: `due_date`, `priority`
- **生まれるルール（不変条件）をentityに閉じ込める**
  - 「期限切れのTodoは完了できない」
  - 「完了済みのTodoは期限を変更できない」
    → `Complete()` / `ChangeDueDate()` メソッドにガードを書く。serviceからバリデーションが消え、entityに移る。
- **Value Objectを導入する**
  - `priority` をただの `int` でなく、`1〜5` の範囲を自己保証する `type Priority struct{...}` にする。
    → 「ただのintをVOにすると何が嬉しいか」を作為的でなく体感できる。
- 学習内容: 不変条件、Value Object、クエリの複雑化、DTO設計

### 拡張案2：タグ機能（Aggregateの入り口 + トランザクション）

- テーブル: `tags`, `todo_tags`（多対多）
- **生まれるルール**: 「1つのTodoにタグは5個まで」のような整合性制約
  → Todoを集約ルート、タグ集合をその内部とみなす **Aggregateの整合性境界**の入り口になる。
- **トランザクションが必要になる**: todoとtodo_tagsを一貫して更新する
  → 生SQLで `Begin / Commit / Rollback` を手で書く。serviceがトランザクション境界を持つ意味を体感。
- 学習内容: 多対多、中間テーブル、Aggregate、トランザクション

### 2周目で「導入を検討」する（＝1周目では入れない）

これらは「必要になったら重くする」の典型例。導入条件をセットで覚える。

- **Value Object** — プリミティブな値に制約・意味が生まれたとき
- **不変条件のentity封じ込め** — 状態間にルールが生まれたとき
- **Aggregate** — 複数entityにまたがる整合性を守る必要が出たとき
- **トランザクション** — 複数テーブルを一貫して更新する必要が出たとき
- **domain service** — どのentityにも属さない、entity跨ぎのルールが出てきたとき

---

## 学習サイクル

```txt
1. 実装する
2. 詰まる
3. 調べる
4. 修正する
5. 「なぜそうなるか」を言語化する
```

---

## 最終ゴール：説明できる状態

- なぜこのレイヤー構成なのか
- なぜこの依存方向なのか
- なぜrepositoryをinterfaceにするのか／なぜ利用側で定義するのか
- なぜビジネスルールをmodelに置くのか
- なぜ生SQLを選んだのか／ORMとのトレードオフは何か
- なぜ net/http 標準で書くのか／フレームワークが何を隠していたか
- なぜ契約（OpenAPI）を先に書くのか／手書きと生成のトレードオフ
- DDDの戦術パターンを「いつ導入するか」の判断軸

---

## 業務コード（クリーンアーキ寄り / DDD）への読み替え

このデファクト版を学び切った後、現在の業務コードを読むための対応表と思考法。
**このデファクト版は「業務アーキを薄く畳んだもの」**であり、業務アーキは「これを目的別にさらに割ったもの」と理解するとよい。

### レイヤーの対応表

| デファクト版（軽量クリーン + DDD-lite）   | 業務版（厳密クリーンアーキ / DDD）          | 何が起きているか                                                           |
| ----------------------------------------- | ------------------------------------------- | -------------------------------------------------------------------------- |
| `handler`                                 | `handler` / `controller` / `presentation`   | ほぼそのまま対応。HTTPの受付役                                             |
| `service`（段取り部分）                   | `usecase`（アプリケーション層）             | serviceの「1操作の段取り」だけを切り出した層。操作の数だけ増える           |
| `service`（ルール部分）                   | `domain service` / entityのメソッド         | serviceの「業務ルール」をドメイン層に寄せたもの                            |
| `model`（振る舞い付き）                   | `entity` / `value object` / `aggregate`     | 学習版で既にメソッドを持たせていたので地続き。業務版ではVO・集約に分かれる |
| `repository` interface（service側に定義） | `domain/repository`（domain層に定義）       | **interfaceの所有者が変わる**（利用側→domain層）。これが最大の差分         |
| `repository` 実装                         | `infrastructure` / `persistence`            | 実装を外側の層に分離。中身（SQL）はほぼ同じ                                |
| `dto`                                     | `dto` / `request`・`response` / `presenter` | ほぼ対応。業務版では入力用・出力用をさらに分けることがある                 |

### 差分マップ：すべては「依存方向」に収束する

デファクト版と厳密版の差は3つあるが、①が原因で②③はその帰結。

#### ① 依存方向 ＝ interfaceを誰が所有するか（最大かつ本質）

- **DIP（具象でなく抽象に依存）は両者で共通**。デファクト版でもserviceは生SQL実装でなくinterfaceに依存している。ここは変わらない。
- 違うのは**interfaceの所有者（住む層）**。
  - デファクト版: interfaceは**利用側（service / アプリ層）**が所有。
  - 厳密版: interfaceは**domain層（最も内側）**が所有し、実装がinfrastructure（外側）。
    → 「外側が内側に従属する」**依存性逆転**が成立。domain層はDBを一切知らない。
- クリーンアーキの "クリーン" が指すのは層の数ではなく、**この矢印の向き**。

#### ② serviceの縦割れ（①の帰結）

- domainを独立した内側の層として立てる（①）から、ルールをそこに置く必要が生じ、serviceが
  `usecase`（段取り）と `domain`（ルール）に割れる。
- 結果、usecaseは薄いオーケストレーターになり、判断ロジックはdomainへ移る。

#### ③ structの詰め替え／マッパー（①の帰結）

- 層を独立させる（①）ため、`entity ⇄ DBレコード ⇄ DTO` を変換する。各層を外側の都合から隔離。
- 「変換コードが多い＝悪」ではなく、**独立性とのトレードオフ**として読む。

### DDDとクリーンアーキの関係

- **クリーンアーキ = 箱（依存方向の構造）／ DDD = 中身（ドメインのモデリング手法）** の別レイヤー。
- DDDの中核要求「ドメインを技術的関心事から隔離して純粋に保つ」と、クリーンアーキの「内側=domainが外側を知らない構造」が同じものを別の言葉で言っている → だから相性が良い。
- ただし**箱だけクリーンアーキで中身は貧血モデル**（DDDしていない）も普通に存在する。箱と中身は独立に効く。

| DDDの構成要素                 | 置かれる層          | 役割                             |
| ----------------------------- | ------------------- | -------------------------------- |
| Entity（同一性を持つ）        | domain              | 振る舞いとルールを持つ           |
| Value Object（値で同一性）    | domain              | 不変・自己検証する値             |
| Aggregate（整合性の境界）     | domain              | 複数entityを束ね不変条件を守る   |
| Domain Service（entity跨ぎ）  | domain              | どのentityにも属さないルール     |
| Repository（永続化の抽象）    | domain（interface） | ドメインから永続化を隠す         |
| Application Service / Usecase | application         | 操作の段取り（ルールは持たない） |

### DDDの深さはグラデーション

```txt
貧血モデル（modelはただの箱）
  ↓  modelにメソッドを持たせる
DDD-lite（= このデファクト版で目指すところ）
  ↓  Value Object / 不変条件 / Aggregate を足す（2周目）
本格DDD
  ↓  ドメインイベント / CQRS / 集約設計の厳密化
フルDDD（重装備）
```

> DDDを深めるほど、ルールがmodelに集まりserviceが薄くなり、構造は自然と厳密クリーンアーキに寄っていく。
> 最後に残る構造的な一歩が **①interfaceの所有権をdomainに移すこと**。

### 読み替えチェックリスト（業務コードを開いたとき）

業務コードのあるファイルを開いたら、次を自問する。

1. これは学習版のどの層に対応するか？（handler / service=usecase+domain / repository / model / dto）
2. **このinterfaceはdomainが所有しているか、利用側が所有しているか？**（→ これ一発でどちらのアーキか判別でき、残りの差分も予測がつく）
3. serviceが usecase と domain に割れているなら、この処理は「段取り」か「ルール」か？
4. struct の詰め替えが起きているなら、何から何への変換で、何を隔離しているのか？
5. modelは振る舞いを持つ（DDD）か、ただの箱（貧血）か？
6. ORM呼び出しは、生SQLでいうと何に相当するか？

---

## 一言まとめ

```txt
契約を先に書く（OpenAPIで外部契約を確定 → それを満たすよう実装）
↓
軽く作る（net/http標準・生SQL・interfaceは利用側定義・層は薄く・modelに振る舞い）
↓
1周目で骨格、2周目でルールを生んでDDD-liteを効かせる
↓
必要になったら重くする（usecase分離・VO・Aggregate・transaction）
↓
判断軸を身につける
↓
業務の厳密クリーンアーキ/DDDを「これを目的別に割ったもの」「①依存方向を逆転させたもの」として読み解く
```
