# Go Todo API 学習ガイド

## 目的

GoでTodo APIを**フルスクラッチ・手書き**で作り、次を身につける。

- `net/http` と `database/sql` だけでWebサーバを組む方法（フレームワーク・ORMは使わない）
- シンプルな3層アーキテクチャの責務分担と依存方向
- Goらしいinterfaceの切り方（利用側が定義し、実装側は暗黙的に満たす）

ゴールは設計の「正解」を暗記することではなく、
**なぜその構造にするのかを自分の言葉で説明できるようになること**。

---

## 基本アーキテクチャ：Feature 単位のシンプルな3層

```txt
Handler → Service → Repository → DB(生SQL)
```

| 層 | 責務 |
| --- | --- |
| Handler | HTTPの入出力。`http.ResponseWriter` / `*http.Request` を触るのはここだけ |
| Service | ユースケース・業務処理。処理の段取りと業務ルール |
| Repository | 永続化。`database/sql` で生SQLを書く |

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
Handler → Service → Repository → DB
```

- 具体型でよい。interfaceは切らない
- 単純なstructでよい。振る舞いを持たせない
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
DBがテストの邪魔になる
外部サービスとの通信が追加される
        ↓
利用側（Service）にinterfaceを定義する
実装（Repository）はそのinterfaceをimportせず暗黙的に満たす
        ↓
依存性逆転・境界分離（Clean Architectureの原則）を学ぶ
```

Goらしいinterfaceのポイントは、**利用側が必要なメソッドだけを宣言する**こと（consumer-defined interface）。実装側はinterfaceの存在を知らなくてよい。

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
