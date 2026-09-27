# Go Todo API 学習ガイド

## 目的

GoでTodo APIを**フルスクラッチ・手書き**で作り、次を身につける。

- `net/http` 標準だけでWebサーバを組む方法（フレームワーク・ORMは使わない。DBを入れるときも `database/sql` で生SQL）
- ヘキサゴナルをベースにした3層の責務分担と依存方向
- Goらしいinterfaceの切り方（利用側が定義し、実装側は暗黙的に満たす）
- 集約（一緒に整合性を守る範囲）を境界にしてパッケージと Service を切る判断

ゴールは設計の「正解」を暗記することではなく、
**なぜその構造にするのかを自分の言葉で説明できるようになること**。

---

## 設計思想

守りたいのは次の3つだけ。形はこれを守るための手段であって目的ではない。

- ライフサイクルが同じものを一箇所に集約する
- 複雑性を局所化する
- 知識を漏らさず、必要最低限のインターフェースで通信する

これを実現する枠組みとしてヘキサゴナル（Ports and Adapters）を採る。主張は「コアは外界を知らない、依存は外から内へ」の1点で、これ自体はタダ。重くなるのは境界を全部明示しようとしたときなので、**境界を明示する費用は、差し替え・テスト・ずれの吸収で回収できる場所にだけ払う**。

構造は複雑さのある場所に置く。Todo 規模では複雑さがどこにも無いので、この構成は儀式になる。それは学習用途として割り切る（各 Phase で「なぜ要るか」を言語化するために、層を先に置く）。逆に複雑さが通信にあるシステム（中継サーバーなど）にはこの構成を当てはめない。

---

## 基本アーキテクチャ：ヘキサゴナルベースの3層 + Feature 単位

```txt
adapter（外界との接続） → feature（コア：ルール + 段取り + 永続化）
```

| 層 | 責務 | 置き場 |
| --- | --- | --- |
| Handler | HTTPの入出力。`http.ResponseWriter` / `*http.Request` を触るのはここだけ | `internal/adapter/http` |
| Service | ユースケースの段取り。Phase 2 までは業務ルールもここ | `internal/<feature>/service.go` |
| Repository | 永続化。最初はインメモリのスライス。DBが必要になったら `database/sql` + SQLite で生SQL | `internal/<feature>/repository.go` |

層はこの3つまで。これ以上は増やさない。

### Feature の粒度

Feature は **一緒に整合性を守る範囲（集約）** で切る。目安は親リソース1つ + そのサブリソース（例: `todo` に `/todos` と `/todos/{id}/comments`）。単独で存在できるリソース（例: `/tags`）は別 Feature。

見分ける質問は2つ。

- A が1つも無くても B は存在できるか
- B の変更は1回の操作で全 A に効くか

両方 Yes なら別集約（別 Feature）、両方 No なら同じ集約の中に収める。

Feature ≒ 集約なので、**Feature 内の Service は基本1つ**。増えるとしたら集約以外の理由（集約を跨ぐ操作、読み取りの分離、ユースケース分解）で、それは Phase 4 で扱う。

### ディレクトリ

```txt
cmd/
  server/
    main.go            # 起動・DB接続・配線（依存注入はここ）
internal/
  adapter/
    http/              # 入力側。リソース単位の TodoHandler / TagHandler、router.go
    worker/            # 入力側。ジョブ・cron（必要になったら）
    <外部名>/          # 出力側。複数 Feature で共有する外部クライアント（必要になったら）
  todo/
    todo.go            # エンティティ。ルールはここ（Phase 3 以降）
    service.go         # 段取り
    repository.go      # 永続化（interface + 実装）
  tag/                 # Phase 3 で追加
api/
  *.http               # 動作確認用リクエスト（Phase 1 はこれが主体）
  openapi.yaml         # 外部契約（Phase 2 以降で正本にする）
```

Repository を Feature の中に置くのは「集約専用だから」。外部 API クライアントのように複数 Feature から共有されるものは `adapter/` に出す。線引きは「外部かどうか」ではなく「集約専用か共有か」。

Repository を `adapter/sqlite` のように外へ出したくなる条件は2つ。SQL の実装が太って Feature の見通しを損ねたとき、あるいは Feature パッケージが DB ドライバを import すること自体を避けたいとき。どちらも「起きたら」でよい。

### 依存ルール

- `adapter → feature` の一方向。feature は adapter を import しない
- feature 間も一方向。相手の公開型・メソッド経由でのみ触る（相手の Repository や内部型に触らない）。双方向依存は import cycle になるので、片方に寄せるか第三のパッケージに出す
- Handler → Repository を直接呼ばない。Handler に SQL を書かない
- Service に `*http.Request` / `http.ResponseWriter` を渡さない
- Repository に HTTP の概念を持ち込まない
- トランザクションを知っていいのは Service まで。Handler とエンティティは知らない
- 集約を跨ぐ操作は「主に変わる側」の Feature の Service に置く
- ルーティングは Go 1.22+ の `http.ServeMux`（`"GET /todos/{id}"` + `r.PathValue`）。`adapter/http/router.go` に集約する
- Service と Repository は同一パッケージ内のファイル分けなので、その境界は規律で守る。ただし Service のフィールドを interface 型で宣言すれば、Service から実装の非公開フィールドには触れなくなる（Phase 2 以降）
- 非公開フィールドは同一パッケージ内でもメソッド経由でしか変えない（Phase 3 以降の規律）。言語で守りたくなったら型だけサブパッケージに出す

### 判断の基本形

迷ったら「戻すのが安い方」を選び、隣のエンドポイントと揃える。

- **外向きはリソース指向で固定する**。URL は名詞、操作は HTTP メソッド。Handler はリソース単位の struct（`TodoHandler`, `TagHandler`）にメソッドを束ねる。これは可読性のためで必須ではない。URL と変更される集約のずれ（`PUT /todos/{id}/tags/{tagId}` は tag ではなく todo を変える）は `adapter/http` が吸収し、Service はユースケースの語彙で書く。状態遷移（完了など）を `PATCH` の属性更新で表すかコントローラーリソース（`POST /todos/{id}/complete`）で表すかは Phase 3 で1度決め、以後は混ぜない
- **ルールは所有するものに置く**。Phase 2 までは Service に直接書く（トランザクションスクリプト）。Phase 3 で `Todo` のメソッドに引き上げる。単一のエンティティに置けないルール（複数の集約を対等に見て決まる判定）は同パッケージの関数にする。Domain Service という型は作らない
- **バリデーションは所有する層で行う**。形式（JSON が壊れている、id が数字でない）は Handler、意味（name が空、期限が過去）はルールの置き場と同じ。Service は Handler が先に弾くことを前提にしない。Service はドメインのエラー（`ErrXxx`）を返し、Handler が `errors.Is` でステータスコードに翻訳する
- **リクエスト型は本文のあるエンドポイントごとに最初から `adapter/http` に切る**
- **レスポンス型は `Todo` を借りる**のが既定。次のどれかに当たったら、そのリソースの全エンドポイントで `adapter/http` に `xxxResponse` を切り、変換関数も同じ場所に置く
  1. `Todo` に外へ出したくないフィールドが増えた
  2. JSON の形が `Todo` の構造と違う（日時の書式、ネスト、一覧と詳細で項目が違う）
  3. `Todo` から `json` タグを外したい
- **Service と Repository は同じ `Todo` を渡す**。DB の列と `Todo` の表現が食い違ったときだけ Repository 内に行用の struct や変換を置く
- **interface は利用側に切る**。差し替えたい・実装を外に出したい・テストで偽物を入れたい、のどれかが起きたら切る。それまでは具体型でよい。実装を Feature の外に出すときは interface とセット（片方だけでは意味が薄い）
- **公開（大文字）は `adapter`・`main`・他 Feature から触るものだけ**。Feature 側は `Service` / `NewService` / `Store` / `NewStore` / `Todo` / `ID`。ヘルパは小文字。Feature 内の型名はパッケージ名で修飾される前提で短くする（`todo.Service`。`todo.TodoService` にしない）。逆に `adapter/http` はパッケージ名が層なので、型名がリソース名を背負う（`TodoHandler`）

---

## 設計手法の位置づけ

Clean Architecture や DDD は**採用するもの**ではなく、**実装上の痛みが出たときに、その問題を解決する道具として必要な部分だけ導入するもの**。「構造」と「問題解決の手法」を混ぜない。

```txt
採用アーキテクチャ:
  ヘキサゴナルベースの3層 + Feature（集約）単位

必要に応じて使う設計手法:
  Clean Architecture（依存性逆転・境界分離）
  DDD（Entityの振る舞い・Value Object・Aggregate）
  Ports / interface
```

Domain Service は「エンティティに置けなかったルールの残り」を置く小さな箱で、うまく設計されていればほぼ空。作るものとして捉えた瞬間にエンティティが空になる（ドメインモデル貧血症）ので、必要になったら関数を1つ置くだけにする。

学習用プロジェクトなので、痛みの自然発生を待たずに**意図的に痛みを作り**、素朴な状態を一度経験した上で導入する。導入前の不便を知らないと、何を解決しているのか分からないため。

---

## 学習フェーズ

### Phase 1：素朴なWeb API

```txt
adapter/http（Handler） → todo.Service → todo.Store → インメモリのスライス
```

- 具体型でよい。interface は切らない
- 単純な struct でよい。振る舞いを持たせない（ルールは Service に直接書く）
- 永続化はインメモリで十分。DB を入れない
- DDD しない。Clean Architecture を意識しない
- Handler は最初から `adapter/http` に置く。Todo だけの段階では Feature 内に置くのと差が無いが、構成を固定するために最初から分ける

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
利用側（Service）に interface を定義する
実装（Repository）はその interface を import せず暗黙的に満たす
        ↓
依存性逆転・境界分離（Clean Architectureの原則）を学ぶ
```

Goらしい interface のポイントは、**利用側が必要なメソッドだけを宣言する**こと（consumer-defined interface）。実装側は interface の存在を知らなくてよい。
インメモリ実装と SQLite 実装（`database/sql` + 生SQL）が同じ interface を満たし、`main.go` の配線だけで差し替えられる状態が最初の到達点。

同じ原則を Handler → Service にも適用する。`adapter/http` 側に `todoService` interface を宣言し、`*todo.Service` の具体型ではなくそれを受け取る。別パッケージなので自然にこの形になり、Phase 4 で Service を分解しても Handler を変えずに済む。

### Phase 3：ドメインを複雑にする

あえてルールを追加する。

- 期限切れの Todo は完了できない
- 完了後は期限を変更できない
- Priority は 1〜5
- タグは1つの Todo に最大5個

```txt
Service にルールが散らばって辛くなる
        ↓
Entity の振る舞い（メソッドにガードを置く）
Value Object（Priority に範囲を自己保証させる）
Aggregate（Todo とタグの整合性境界）
Entity に置けないルールは同パッケージの関数
        ↓
DDD の必要性を学ぶ
```

Tag の扱いで集約の境界を決める。

- タグを「その Todo に付いたラベル文字列」にするなら `Todo` の値オブジェクトで、`/tags` は不要、Tag Feature も不要
- タグを「アプリ全体で共有される名前付きの実体」（先に作る、リネームすると全 Todo に効く、どの Todo にも付いていなくても存在できる）にするなら別集約・別 Feature

学習では後者を選ぶ。前者だと集約を跨ぐ演習が消えるため。両方の違いを説明できることが目標。

Tag を別 Feature にした時点で出る痛み。

- モジュール内に集約が2つある状態になる。Repository が2つ、トランザクション境界が2つ、集約同士は ID でしか繋がらない（`Todo` は `[]tag.ID` を持つが `[]tag.Tag` は持たない）
- Feature 間の依存方向を決める。`todo → tag` の一方向。tag は todo を知らない
- URL と変更される集約がずれる。`PUT /todos/{id}/tags/{tagId}` は URL 末尾が tags だが変わるのは Todo なので `todo.Service.AttachTag`。`tag.Service` にこのメソッドは生えない
- 読み取りが跨ぐ。`GET /tags/{id}/todos` は tag から todo への参照になるので、`GET /todos?tag={id}` として todo 側に置くか、跨ぐ操作が増えたら第三パッケージへ
- `DELETE /tags/{id}` で付与済み Todo から参照を外すかどうかを決める。外すなら2集約を跨ぐ操作になり、Phase 4 の題材になる。参照を残して表示時に無視する、と決めるなら `tag.Service.Delete` だけで済む。この判断自体がドメイン設計

### Phase 4：Service を分解する

```txt
Phase 3 まで: 段取りとルールが同居するので todo.Service に束ねるのが自然
        ↓
ルールが Entity に移り、Service には段取りだけが残る（この状態を usecase と呼ぶ）
        ↓
束ねる理由が消えたので、1ユースケース1型に分解してもよい（CreateTodo.Do 等）
複数集約を跨ぐ操作の置き場を決める（例: Tag 削除時に Todo から外す）
読み取り系は Query として分けてもよい（変更理由が違う）
```

Service を usecase と domain service に「割る」のではない。ルールが Entity に出ていった後の Service を usecase と呼び直すだけで、層は増えない。Domain Service は Entity に置けないルールが実際に出たときに関数を1つ置く。

跨ぐ操作の置き場は「主に変わる側」の Feature。トランザクションはその Service が開き、2つの Repository に渡す。

ここまで来た形が、業務コードのクリーンアーキテクチャ / DDD をさらに目的別に割ったものの原型になる。

### Phase 5（終幕）：図書館貸出

Todo で固めた構成をそのまま横に増やし、Todo では出なかった痛みを全部出す。

**集約とルール**

- `book`：蔵書。同じタイトルが複数冊。「在庫 0 なら貸出不可」
- `member`：会員。「延滞中は新規貸出不可」「停止中は貸出不可」「5冊まで」
- `loan`：貸出。「返却期限 14 日」「延長は1回まで、延滞後は不可」。状態は 貸出中 → 返却済み / 延滞

**リソース**

```txt
POST   /books               GET /books/{id}
POST   /members             GET /members/{id}
POST   /loans               # 跨ぐ操作。member の状態確認 + book 在庫減 + loan 作成を 1 tx
POST   /loans/{id}/return   # 跨ぐ操作。loan 状態変更 + book 在庫戻し
POST   /loans/{id}/extend
```

**adapter**

- `adapter/http`：上のハンドラ
- `adapter/worker`：日次で延滞検知して `loan` を延滞に遷移、`member` に反映。HTTP 起点でない入力側アダプタ
- `adapter/mail`（または `slack`）：延滞通知。`loan` 側で `Notifier` interface を宣言し、adapter が満たす

**痛みが出る順**

1. 「5冊まで」「延滞中は不可」を `POST /loans` の Service に書くと `member` の内部を覗きたくなる → `member.CanBorrow()` に引き上げる
2. 貸出で book と loan の2つの Repository を触る → トランザクションを Service が開く形を決める
3. 「返却時に在庫戻し」をどの Feature に置くか → `loan → book, member` の一方向で `loan` に置く
4. 延滞検知は HTTP 起点でない → worker が `loan.Service` を叩く入力側アダプタになる

`book` と `member` を CRUD だけで先に作り、`loan` を足した時点で 1〜3 が一気に出る順序で進める。

---

## 学習サイクル

各 Phase で Before / After を比較する。

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
- OpenAPI は手書きし、構文・網羅性・整合性の補正は AI に任せる。ただし直すことより「何がまずいか」のフィードバックが主目的
- GUI は作らず `.http` を主体にする。状態確認が面倒になったら Swagger UI や AI 生成のテスト GUI を検討する

---

## 補足：実務に持ち帰るとき

学習では層を最初から固定するが、実務で小さく始めるなら Service は後から生やしてよい。Handler が Repository とエンティティを直接呼び、段取りが2つ目のエンドポイントと重複した時点で関数に括り出し、それが溜まった時点で Service と呼ぶ。Feature の中で閉じている限り adapter に影響しないので、この構成と矛盾しない。

interface も同じで、差し替えたい・テストしたい・実装を外に出したい、のどれかが起きるまでは具体型でよい。学習で体験した「要る瞬間」を、実務では判断基準として使う。