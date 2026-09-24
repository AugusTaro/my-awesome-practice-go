# Go Todo API 学習ガイド

## 目的

GoでTodo APIを**フルスクラッチ・手書き**で作り、次を身につける。

- `net/http` 標準だけでWebサーバを組む方法（フレームワーク・ORMは使わない。DBを入れるときも `database/sql` で生SQL）
- シンプルな3層アーキテクチャの責務分担と依存方向
- Goらしいinterfaceの切り方（利用側が定義し、実装側は暗黙的に満たす）

ゴールは設計の「正解」を暗記することではなく、
**なぜその構造にするのかを自分の言葉で説明できるようになること**。

---

## 基本アーキテクチャ：Feature 単位のシンプルな3層

```txt
Handler → Service → Repository → 保存先（インメモリ → SQLite）
```

| 層 | 責務 |
| --- | --- |
| Handler | HTTPの入出力。`http.ResponseWriter` / `*http.Request` を触るのはここだけ |
| Service | ユースケース・業務処理。処理の段取りと業務ルール |
| Repository | 永続化。最初はインメモリのスライスで十分。DBが必要になったら `database/sql` + SQLite で生SQL |

ディレクトリは Feature 単位でまとめる。

```txt
cmd/
  server/
    main.go          # 起動・DB接続・ServeMux配線（依存注入はここ）
internal/
  todo/
    handler.go
    service.go
    repository.go
    todo.go          # Todo構造体
api/
  *.http             # 動作確認用リクエスト（Phase 1 はこれが主体）
  openapi.yaml       # 外部契約（Phase 2 以降で正本にする）
```

### 依存ルール

- Handler → Repository を直接呼ばない。HandlerにSQLを書かない
- Serviceに `*http.Request` / `http.ResponseWriter` を渡さない
- RepositoryにHTTPの概念を持ち込まない
- ルーティングは Go 1.22+ の `http.ServeMux`（`"GET /todos/{id}"` + `r.PathValue`）で書く
- 層は同一パッケージ内のファイル分けなので、上の境界はコンパイラでなく規律で守る。隣の層との接点は「相手のメソッド一覧」、Phase 2 以降は利用側に置いた interface で読む

### 判断の基本形

迷ったら「戻すのが安い方」を選び、隣のエンドポイントと揃える。

- **外向きはリソース指向で固定する**。URL は名詞、操作は HTTP メソッド。ユースケースとのズレは Handler が吸収し、Service はユースケースの語彙で書く。状態遷移（完了など）を `PATCH` の属性更新で表すかコントローラーリソース（`POST /todos/{id}/complete`）で表すかは Phase 3 で1度決め、以後は混ぜない
- **バリデーションは所有する層で行う**。形式（JSON が壊れている、id が数字でない）は Handler、意味（name が空、期限が過去）は `Todo` のメソッドで、Service がそれを呼ぶ。Service は Handler が先に弾くことを前提にしない。Service はドメインのエラー（`ErrXxx`）を返し、Handler が `errors.Is` でステータスコードに翻訳する
- **レスポンス型は `Todo` を借りる**のが既定。次のどれかに当たったら、そのリソースの全エンドポイントで `handler.go` に `xxxResponse` を切り、変換関数も `handler.go` に置く
  1. `Todo` に外へ出したくないフィールドが増えた
  2. JSON の形が `Todo` の構造と違う（日時の書式、ネスト、一覧と詳細で項目が違う）
  3. `Todo` から `json` タグを外したい
- **Service と Repository は同じ `Todo` を渡す**。DB の列と `Todo` の表現が食い違ったときだけ Repository 内に行用の struct や変換を置く
- **公開（大文字）は `main` から触るものだけ**。`Handler` / `NewHandler` / `Store` / `NewStore` / `Todo`。リクエスト型やヘルパは小文字。型名はパッケージ名で修飾される前提で短くする（`todo.Handler`。`todo.TodoHandler` にしない）

---

## 設計手法の位置づけ

Clean Architecture や DDD は**採用するもの**ではなく、**実装上の痛みが出たときに、その問題を解決する道具として必要な部分だけ導入するもの**。「構造」と「問題解決の手法」を混ぜない。

```txt
採用アーキテクチャ:
  3層 + Feature単位

必要に応じて使う設計手法:
  Clean Architecture（依存性逆転・境界分離）
  DDD（Entityの振る舞い・Value Object・Aggregate・Domain Service）
  Ports / interface
```

学習用プロジェクトなので、痛みの自然発生を待たずに**意図的に痛みを作り**、素朴な状態を一度経験した上で導入する。導入前の不便を知らないと、何を解決しているのか分からないため。

---

## 学習フェーズ

### Phase 1：素朴なWeb API

```txt
Handler → Service → Repository → インメモリのスライス
```

- 具体型でよい。interfaceは切らない
- 単純なstructでよい。振る舞いを持たせない
- 永続化はインメモリで十分。DBを入れない
- DDDしない。Clean Architectureを意識しない

機能は Todo の CRUD。

```txt
POST   /todos
GET    /todos
GET    /todos/{id}
PATCH  /todos/{id}
DELETE /todos/{id}
```

### Phase 2：抽象化する理由を作る

```txt
インメモリを SQLite に差し替えたい（Service を触らずに）
DBがテストの邪魔になる
外部サービスとの通信が追加される
        ↓
利用側（Service）にinterfaceを定義する
実装（Repository）はそのinterfaceをimportせず暗黙的に満たす
        ↓
依存性逆転・境界分離（Clean Architectureの原則）を学ぶ
```

Goらしいinterfaceのポイントは、**利用側が必要なメソッドだけを宣言する**こと（consumer-defined interface）。実装側はinterfaceの存在を知らなくてよい。
インメモリ実装と SQLite 実装（`database/sql` + 生SQL）が同じinterfaceを満たし、`main.go` の配線だけで差し替えられる状態が最初の到達点。

### Phase 3：ドメインを複雑にする

あえてルールを追加する。

- 期限切れのTodoは完了できない
- 完了後は期限を変更できない
- Priority は 1〜5
- タグは1つのTodoに最大5個

```txt
Serviceにルールが散らばって辛くなる
        ↓
Entityの振る舞い（メソッドにガードを置く）
Value Object（Priorityに範囲を自己保証させる）
Aggregate（Todoとタグの整合性境界）
Domain Service（どのEntityにも属さないルール）
        ↓
DDDの必要性を学ぶ
```

### Phase 4：Serviceを分解する

```txt
Service = 段取り + 業務ルール
        ↓
UseCase  ← 処理の段取り
Domain   ← 業務ルール
```

ここまで来た形が、業務コードのクリーンアーキテクチャ / DDD をさらに目的別に割ったものの原型になる。

---

## 学習サイクル

各Phaseで Before / After を比較する。

```txt
単純に作る
    ↓
困る
    ↓
問題を言語化する
    ↓
解決する設計手法を知る
    ↓
導入する
    ↓
Before / After を比較する
```

---

## API契約と動作確認

- Phase 1 はコードを書いて体感することを優先する。`api/*.http` / curl で叩きながら進め、OpenAPI は後追いで書いてもよい
- Phase 2 以降、外部から見た振る舞いを固定したくなった時点で `api/openapi.yaml` を正本にする（理想はスキーマファーストだが、Phase 1 では強制しない）
- OpenAPIは手書きし、構文・網羅性・整合性の補正はAIに任せる。ただし直すことより「何がまずいか」のフィードバックが主目的
- GUIは作らず `.http` を主体にする。状態確認が面倒になったら Swagger UI や AI生成のテストGUIを検討する
