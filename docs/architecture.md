# Go Todo API 学習ガイド

## 目的

GoでTodo APIを**フルスクラッチ・手書き**で作り、次を身につける。

- `net/http` 標準だけでWebサーバを組む方法（フレームワーク・ORMは使わない。DBを入れるときも `database/sql` で生SQL）
- ヘキサゴナルの最小形（入口と出口の2境界）と、痛みが出た場所にだけ層を足していく判断
- Goらしいinterfaceの切り方（利用側が定義し、実装側は暗黙的に満たす。必要になるまで切らない）
- 集約（一緒に整合性を守る範囲）を境界にしてパッケージ・Repository・Service を切る判断

ゴールは設計の「正解」を暗記することではなく、
**なぜその構造にするのかを自分の言葉で説明できるようになること**。

---

## 設計思想

守りたいのは次の3つだけ。形はこれを守るための手段であって目的ではない。

- ライフサイクルが同じものを一箇所に集約する
- 複雑性を局所化する
- 知識を漏らさず、必要最低限のインターフェースで通信する

これを実現する枠組みとしてヘキサゴナル（Ports and Adapters）を採る。主張は「コアは外界を知らない、依存は外から内へ」の1点で、これ自体はタダ。重くなるのは境界を全部明示しようとしたときなので、**境界を明示する費用は、差し替え・テスト・ずれの吸収で回収できる場所にだけ払う**。

最初から固定するのは、**後から変えると高いもの**だけ。HTTP 境界（`adapter/web`）、永続化の境界（Repository を集約単位で分ける）、パッケージの依存方向。これが最低ラインの「2境界」。Service・interface・型の束ね方は後から変えても安いので、**そうしないとしんどい、と思った瞬間に足す**。学習では、その瞬間を各 Phase で意図的に作る。

構造は複雑さのある場所に置く。複雑さが通信にあるシステム（中継サーバーなど）にはこの構成を当てはめない。

---

## 基本アーキテクチャ：2境界から始めるヘキサゴナル + Feature 単位

```txt
adapter（外界との接続） → feature（コア：ルール + 段取り + 永続化）
```

| 層 | 責務 | 置き場 | いつから |
| --- | --- | --- | --- |
| Handler | HTTPの入出力。形式バリデーション。ドメインエラー → ステータスコード翻訳。Service が無い間は段取りもここ | `internal/adapter/web` | Phase 1 |
| Repository | 集約の保存・取得。SQL はここだけ。最初はインメモリのスライス、Phase 2 で `database/sql` + SQLite | `internal/<feature>/repository.go` | Phase 1 |
| エンティティ | 状態遷移と不変条件 | `internal/<feature>/<集約>.go` | Phase 1 は素の struct、Phase 3 でメソッドを持つ |
| Service | ユースケースの段取り。tx の開始と終了。複数 Repository の呼び分け | `internal/<feature>/service.go` | **Phase 3**（tx を跨ぐ操作が出たとき） |

層は最大でこの3つ（Handler / Service / Repository）。これ以上は増やさない。

### Feature の粒度

Feature は **業務領域** で切る。一つの業務が語れる範囲で、集約を複数含む。集約単位でパッケージを切るのは細かすぎる（Todo と Tag は同じ `todo/` に入る）。

集約の見分け方は2つの質問。

- A が1つも無くても B は存在できるか
- B の変更は1回の操作で全 A に効くか

両方 Yes なら別集約（型を分け、Repository も分ける）、両方 No なら同じ集約の中に収める。

**Repository は最初から集約ごとに分ける**。Service は Feature に基本1つで、集約ごとに割るのは不都合が出てから。Repository を束ねると「1回の Save で複数集約を書く」実装が紛れ込むので、こちらだけは先に分けておく。

### ディレクトリ

```txt
cmd/
  server/
    main.go            # 起動・DB接続・配線（依存注入はここ）
internal/
  adapter/
    web/               # 入力側（HTTP）。リソース単位の TodoHandler / TagHandler、router.go
    worker/            # 入力側。ジョブ・cron（Phase 5）
  todo/
    todo.go            # 集約1。Phase 3 からルールをメソッドで持つ
    tag.go             # 集約2（Phase 3 で追加）
    repository.go      # 集約ごとの Repository。実装も同居
    service.go         # Phase 3 で登場。段取りと tx
api/
  *.http               # 動作確認用リクエスト（Phase 1 はこれが主体）
  openapi.yaml         # 外部契約（Phase 2 以降で正本にする）
```

Repository を Feature の中に置くのは「集約専用だから」。複数 Feature から共有される外部クライアントは `adapter/` に出す。Repository を `adapter/sqlite` に出したくなる条件（Feature が増えて配管を共有したい、ドメインを純粋に保ちたい）は Phase 5 で扱う。

### 依存ルール

- `adapter → feature` の一方向。feature は adapter を import しない
- Service が無い間（Phase 1〜2）はハンドラが Repository を直接呼ぶ。**Service を切った操作については** Handler → Repository を直接呼ばない
- Handler に SQL を書かない。Repository に HTTP の概念を持ち込まない。Service に `*http.Request` / `http.ResponseWriter` を渡さない
- トランザクションを知っていいのは Service まで。Handler とエンティティは知らない（Service が無い間は tx を跨ぐ操作が存在しない、が前提。跨いだら Service を切る）
- ルーティングは Go 1.22+ の `http.ServeMux`（`"GET /todos/{id}"` + `r.PathValue`）。`adapter/web/router.go` に集約する
- 同一パッケージ内の境界（Service と Repository、エンティティの非公開フィールド）は規律で守る。Phase 3 以降、非公開フィールドはメソッド経由でしか変えない

### 判断の基本形

迷ったら「戻すのが安い方」を選び、隣のエンドポイントと揃える。

- **外向きはリソース指向で固定する**。URL は名詞、操作は HTTP メソッド。Handler はリソース単位の struct（`TodoHandler`, `TagHandler`）にメソッドを束ねる（可読性のためで必須ではない）。URL と変更される集約のずれは `adapter/web` が吸収する。状態遷移（完了など）を `PATCH` の属性更新で表すかコントローラーリソース（`POST /todos/{id}/complete`）で表すかは Phase 3 で1度決め、以後は混ぜない
- **Service を切る条件**（どれか一つ）：1操作で複数の Repository を1トランザクションで触る／入口が2つになった（HTTP に加えて worker 等）／段取りそのものにルールがある（作成後に通知、失敗したら取り消し）／2つのハンドラが同じ段取りを書き始めた。一言で言えば「ハンドラに、読む・検証する・呼ぶ・書く、以外の行が出たら」
- **ルールは所有するものに置く**。Phase 2 まではハンドラに直接書く（トランザクションスクリプト）。Phase 3 で `Todo` のメソッドに引き上げる。単一エンティティに置けないルールは同パッケージの関数。Domain Service という型は作らない
- **バリデーションは所有する層で行う**。形式（JSON が壊れている、id が数字でない）は Handler。意味（name が空、期限が過去）はルールの置き場と同じ。内側は外側が先に弾くことを前提にしない。エンティティ・Repository・Service はドメインのエラー（`ErrXxx`）を返し、Handler が `errors.Is` でステータスコードに翻訳する
- **リクエスト型は本文のあるエンドポイントごとに最初から `adapter/web` に切る**
- **レスポンス型は `Todo` を借りる**のが既定。次のどれかに当たったら、そのリソースの全エンドポイントで `adapter/web` に `xxxResponse` を切り、変換関数も同じ場所に置く
  1. `Todo` に外へ出したくないフィールドが増えた
  2. JSON の形が `Todo` の構造と違う（日時の書式、ネスト、一覧と詳細で項目が違う）
  3. `Todo` から `json` タグを外したい
- **Handler / Service と Repository は同じ `Todo` を渡す**。DB の列と `Todo` の表現が食い違ったときだけ Repository 内に行用の struct や変換を置く
- **interface は利用側に切り、必要になるまで切らない**。切る理由は3つだけ：依存の向き（利用側が実装のパッケージを import したくない）／複数の本物の実装（インメモリは本物として数える）／テストのフェイク。「将来増えるかも」「密結合を防ぐ」は理由にならない（後者は非公開フィールドで既に達成されている）。Repository を Feature 内に置いている間は動機がほぼ無く、実装を `adapter/sqlite` に出すときに cycle により強制になる
- **公開（大文字）は `adapter`・`main`・他 Feature から触るものだけ**。Feature 側は `Store` / `NewStore` / `Todo` / `ID`、Phase 3 以降 `Service` / `NewService`。ヘルパは小文字。Feature 内の型名は短く（`todo.Store`。`todo.TodoStore` にしない）。`adapter/web` はパッケージ名が層なので型名がリソース名を背負う（`TodoHandler`）

---

## 設計手法の位置づけ

Clean Architecture や DDD は**採用するもの**ではなく、**実装上の痛みが出たときに、その問題を解決する道具として必要な部分だけ導入するもの**。「構造」と「問題解決の手法」を混ぜない。

```txt
最初から固定するもの（後から変えると高い）:
  adapter/web の分離、Repository を集約単位で分ける、パッケージの依存方向

痛みが出たら足すもの（後から変えても安い）:
  Service、interface、エンティティのメソッド、Service の分割、Query の分離
```

Domain Service は「エンティティに置けなかったルールの残り」を置く小さな箱で、うまく設計されていればほぼ空。作るものとして捉えた瞬間にエンティティが空になる（ドメインモデル貧血症）ので、必要になったら関数を1つ置くだけにする。

学習用プロジェクトなので、痛みの自然発生を待たずに**意図的に痛みを作り**、素朴な状態を一度経験した上で導入する。導入前の不便を知らないと、何を解決しているのか分からないため。**Service も同じで、Phase 3 で tx を跨ぐ操作が出て初めて切る。** 痛みが出て何かを足したら「何が痛かったか、何を足したか、事前に切っておくべきだったか」を一行メモする。

---

## 学習フェーズ

### Phase 1：2境界だけの素朴なWeb API

```txt
adapter/web（Handler） → todo.Store → インメモリのスライス
```

- Service は無い。ハンドラが Store を直接呼び、段取りもルールもハンドラに書く
- 具体型でよい。interface は切らない。Handler は `*todo.Store` を受け取る
- 単純な struct でよい。振る舞いを持たせない
- 永続化はインメモリで十分。DB を入れない
- DDD しない。Clean Architecture を意識しない

機能は Todo の CRUD。

```txt
POST   /todos
GET    /todos
GET    /todos/{id}
PATCH  /todos/{id}
DELETE /todos/{id}
```

到達点：ハンドラが「読む → 検証する → Store を呼ぶ → 書く」の4手で書けていること。ここで Service を切りたくなる瞬間が来ないことを確認する。

### Phase 2：抽象化する理由を作る

```txt
インメモリを SQLite に差し替えたい（ハンドラを触らずに）
        ↓
利用側（ここでは Handler）に interface を定義する
実装（Store）はその interface を import せず暗黙的に満たす
        ↓
依存性逆転・境界分離を学ぶ
```

Goらしい interface のポイントは、**利用側が必要なメソッドだけを宣言する**こと（consumer-defined interface）。実装側は interface の存在を知らなくてよい。Service が無いので利用側はハンドラで、interface は `adapter/web` 側に非公開で宣言する。

インメモリ実装と SQLite 実装（`database/sql` + 生SQL）が同じ interface を満たし、`main.go` の配線だけで差し替えられる状態が到達点。同じテストスイートを両実装に流す（契約テスト）。

Go では interface の後付けが実装に対してゼロコストなので、Phase 1 で切らなかったことに何の負債も無いことをここで確認する。また、ここで切った interface は本番 SQLite なら実はテストにも要らない（インメモリモードの本物を差せる）ことも確認する。切ったのは「切り方を学ぶため」と「Phase 5 で実装を外に出す準備」。

### Phase 3：ドメインを複雑にし、Service が生まれる

あえてルールを追加する。

- 期限切れの Todo は完了できない
- 完了後は期限を変更できない
- Priority は 1〜5
- タグは1つの Todo に最大5個

Tag は「アプリ全体で共有される名前付きの実体」（先に作る、リネームすると全 Todo に効く、どの Todo にも付いていなくても存在できる）として **同じ `todo` パッケージに2つ目の集約** として追加する。値オブジェクト（Todo に付いたラベル文字列）にする案との違いを説明できることが目標。

**痛みが出る順**

1. `PUT /todos/{id}/tags/{tagId}` で Tag の存在確認と Todo の更新、2つの Repository を1操作で触る。tx をハンドラに書きたくない → **ここで初めて `todo.Service` を切る**。段取りをハンドラから Service に移す
2. ルールが Service（と残ったハンドラ）に散って辛くなる → エンティティのメソッドにガードを置く（`todo.Complete()`、`todo.AttachTag(tagID)`）。Value Object（`Priority` に範囲を自己保証させる）
3. URL と変更される集約がずれる。`PUT /todos/{id}/tags/{tagId}` は URL 末尾が tags でも変わるのは Todo なので `Service.AttachTag` は Todo 側の操作。Tag Repository は読むだけ
4. `DELETE /tags/{id}` で付与済み Todo から参照を外すかどうかを決める。外すなら2集約を跨ぐ操作で Service の仕事。参照を残して表示時に無視すると決めるなら Tag Repository の Delete だけで済む。この判断自体がドメイン設計
5. 同一パッケージ内で集約の境界を規律で守る。Todo は `[]tag.ID` を持つが `[]Tag` は持たない。非公開フィールドはメソッド経由でしか変えない

Service を切ったら、Phase 2 の interface の置き場も見直す。利用側が Service になるので `todo` パッケージ内に `Repository` interface を持ち、Handler は Service の具体型か狭い interface を受け取る。

### Phase 4：Service を分解する

```txt
Phase 3 で切った Service には段取りとルールが同居している
        ↓
ルールが Entity に移り、Service には段取りだけが残る（この状態を usecase と呼ぶ）
        ↓
束ねる理由が消えたので、1ユースケース1型に分解してもよい（CreateTodo.Do 等）
読み取り系は Query として分けてもよい（変更理由が違う）
```

Service を usecase と domain service に「割る」のではない。ルールが Entity に出ていった後の Service を usecase と呼び直すだけで、層は増えない。Domain Service は Entity に置けないルールが実際に出たときに関数を1つ置く。

Service を集約ごとに割る（`TodoService` / `TagService`）かどうかもここで判断する。メソッドが増えて見通しが悪い、どの集約の操作かを型で明示したい、のどちらかが起きていなければ割らない。

### Phase 5（終幕）：図書館貸出

Todo で固めた構成を業務領域3つに増やし、Todo では出なかった痛みを全部出す。

**領域と集約**

- `catalog`：蔵書。同じタイトルが複数冊。「在庫 0 なら貸出不可」
- `membership`：会員。「延滞中は新規貸出不可」「停止中は貸出不可」「5冊まで」
- `lending`：貸出。「返却期限 14 日」「延長は1回まで、延滞後は不可」。状態は 貸出中 → 返却済み / 延滞

**リソース**

```txt
POST   /books               GET /books/{id}
POST   /members             GET /members/{id}
POST   /loans               # 跨ぐ操作。member の状態確認 + book 在庫減 + loan 作成を 1 tx
POST   /loans/{id}/return   # 跨ぐ操作。loan 状態変更 + book 在庫戻し
POST   /loans/{id}/extend
```

**adapter**

- `adapter/web`：上のハンドラ
- `adapter/worker`：日次で延滞検知して `loan` を延滞に遷移、`member` に反映。HTTP 起点でない入力側アダプタ。**入口が2つになる**ので、Service が無ければここで切ることになる
- `adapter/mail`（または `slack`）：延滞通知。`lending` 側で `Notifier` interface を宣言し、adapter が満たす

**痛みが出る順**

1. 「5冊まで」「延滞中は不可」を `POST /loans` に書くと `member` の内部を覗きたくなる → `member.CanBorrow()` に引き上げる
2. 貸出で catalog と lending の2つの Repository を触る → tx を `lending.Service` が開く
3. 「返却時に在庫戻し」をどの領域に置くか → `lending → catalog, membership` の一方向で `lending` に置く
4. 延滞検知は HTTP 起点でない → worker が `lending.Service` を叩く
5. 3領域それぞれに Repository があり、接続・tx 管理・行マッピング・エラー変換が重複する → **Repository を `adapter/sqlite` に集める**。このとき Feature 側の `Repository` interface が cycle により必須になる。前に「切らなくてよい」と判断した interface が「切らざるを得ない」ものとして現れることを確認する

`catalog` と `membership` を CRUD だけで先に作り、`lending` を足した時点で 1〜4 が一気に出る順序で進める。**コードを書く前にスキーマを書き、投げるクエリを列挙し、ルールを制約（部分一意索引・CHECK・FK）に落とす**ところから始める。

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
    ↓
「事前に切っておくべきだったか」を一行メモする
```

---

## テスト

「何に確信が欲しいか」で3本。モック中心の中間層テストは、段取りにルールが生まれるまで書かない。

- **ドメインのルール**（エンティティのメソッド・純粋関数）：外部依存なしの単体テスト。Phase 3 以降、ここが一番厚くなる
- **Repository**：本物の SQLite（インメモリ）に対して。インメモリ実装があれば同じスイートを両方に流す
- **HTTP の契約**：`httptest` でサーバ全体を立て、本物の Store を差して端から端まで。エンドポイントごとに正常系1本 + エラー翻訳。Service が無い Phase 1〜2 では Repository 依存のテストはここに吸収される

---

## API契約と動作確認

- Phase 1 はコードを書いて体感することを優先する。`api/*.http` / curl で叩きながら進め、OpenAPI は後追いで書いてもよい
- Phase 2 以降、外部から見た振る舞いを固定したくなった時点で `api/openapi.yaml` を正本にする（理想はスキーマファーストだが、Phase 1 では強制しない）
- OpenAPI は手書きし、構文・網羅性・整合性の補正は AI に任せる。ただし直すことより「何がまずいか」のフィードバックが主目的
- GUI は作らず `.http` を主体にする。状態確認が面倒になったら Swagger UI や AI 生成のテスト GUI を検討する

---

## 補足：実務に持ち帰るとき

学習と実務で始め方は同じ。ハンドラが Repository とエンティティを直接呼び、段取りが2つ目のエンドポイントと重複した時点で関数に括り出し、tx を跨ぐか入口が2つになった時点で Service と呼ぶ。Feature の中で閉じている限り adapter に影響しないので、この構成と矛盾しない。

違うのは組織の前提だけ。将来まとまってリファクタリングする機会が見込めない組織なら、最初から構造を作って守らせる方が結果的に安い。ただしその場合でも先に作るべきは「後から変えると高いもの」（依存方向、`adapter/` の受け皿、Repository を集約単位で分けること）だけで、Service・interface・型の束ね方は組織の性質に関わらず後回しでよい。保険をかける対象も可逆性で選ぶ。