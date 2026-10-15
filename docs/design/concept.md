# dbfolio プロジェクト設計メモ

更新日: 2026-10-04

## 1. プロジェクト概要

**dbfolio** は、データベーススキーマ情報から、ブラウザーで閲覧しやすい静的 HTML ドキュメントを生成する Go 製 CLI ツールを目指す。

初期段階では DB に直接接続せず、`tbls` が生成する `schema.json` を入力とする。

```text
Database
   ↓
  tbls
   ↓
schema.json
   ↓
 dbfolio
   ↓
Static HTML + SVG
```

役割分担は明確にする。

- **tbls**: DB 接続、DB ごとの差異吸収、スキーマ解析、`schema.json` 生成
- **dbfolio**: スキーマを「人が読みやすく閲覧する」ための HTML / ER 図生成

dbfolio は既存ツールの移植や再現ではなく、**tbls のスキーマデータを入力にした、閲覧 UX に特化したデータベースドキュメントジェネレーター**として設計する。

---

## 2. なぜ作るのか

### tbls

tbls は DB 解析の土台として優れている。

- Go 製
- 多数の DB に対応
- DB コメント対応
- Markdown / DOT / Mermaid / JSON / YAML / Excel 等へ出力可能
- `schema.json` を正式に出力可能
- 出力した JSON を再び datasource として読み込み可能
- FK だけでなく追加 Relation / Virtual Relation も扱える
- lint / diff / coverage などが充実

一方、標準のドキュメントは Markdown 中心であり、「HTML をたどりながら DB 全体を見る」用途とは少し方向が異なる。

### dbfolio の狙い

```text
tbls の DB 解析能力
        +
HTML をたどって DB 全体を把握できる閲覧 UX
        ↓
     dbfolio
```

DB 解析機能を二重実装せず、閲覧体験に集中する。

---

## 3. プロジェクト名

プロジェクト名は **dbfolio** とする。

意味:

```text
DB + folio
```

データベース全体を一つの閲覧可能なドキュメントとしてまとめるイメージ。

想定説明文:

> dbfolio — Generate browsable database documentation from your schema.

または tbls 連携を強調する場合:

> dbfolio — Generate browsable static HTML database documentation from tbls schema.json.

将来的に入力形式を増やしても名前を変更する必要がない点も利点。

---

## 4. 初期スコープ

### 入力

初期バージョンは **tbls の `schema.json` のみ**。

生成例:

```bash
tbls out -t json -o schema.json
```

または `tbls doc` がドキュメントと同じディレクトリへ出力する `schema.json` を利用できる。

### 出力

静的 HTML と生成済み SVG。

```text
dbfolio-docs/
├── index.html            ← Overview + テーブル一覧
├── relationships.html
├── tables/
│   ├── users.html
│   ├── orders.html
│   └── ...
├── diagrams/
│   ├── schema.svg
│   ├── users.svg
│   └── ...
└── assets/
    ├── app.css
    └── app.js
```

Web サーバーや JavaScript フレームワークを必須にせず、基本は静的ファイルだけで閲覧できる形を目指す。

テーブル一覧は Overview (`index.html`) に含め、独立した `tables.html` は作らない (Phase 2 で決定)。ページ間のリンクは相対パスにする (`file://` で開けるようにするため)。

---

## 5. tbls schema.json を利用する理由

`tbls` は JSON を正式な出力形式として提供している。

```bash
tbls out -t json -o schema.json
```

さらに `schema.json` を datasource として再利用できる。

```bash
tbls doc json:///path/to/schema.json
```

そのため、`schema.json` は単なる tbls 内部の一時形式ではなく、外部連携に利用できるスキーマ表現と考えられる。

dbfolio 側では、DB 固有の以下の処理を原則として持たない。

- PostgreSQL の `pg_catalog` 解析
- MySQL の `information_schema` 解析
- SQL Server 固有メタデータ取得
- SQLite 固有処理
- DB ドライバー管理
- DB ごとの FK / Index / View 差異吸収

これらは tbls に任せる。

---

## 6. schema.json から利用したい情報

少なくとも以下を利用する。

### Database / Schema

- DB 名
- スキーマ情報
- コメント / description

### Table

- テーブル名
- コメント
- カラム
- Index
- Constraint
- Trigger
- View 等の種別情報

### Column

- 名前
- データ型
- nullable
- default
- comment
- PK / FK に関する情報

### Constraint

- PRIMARY KEY
- FOREIGN KEY
- UNIQUE 等
- 参照先テーブル
- 参照先カラム
- 複合 FK

### Relation

`schema.json` の relation 情報を ER 図・親子関係表示に利用する。

想定する情報:

```text
child table
child columns
parent table
parent columns
cardinality
parent cardinality
definition
virtual relation
```

FK 制約が DB 上に存在しない場合でも、tbls 側で追加 Relation / Virtual Relation が定義されていれば dbfolio で表示できる設計にする。

---

## 7. DB コメント

tbls は DB コメントをドキュメントへ反映できる。

PostgreSQL の例:

```sql
COMMENT ON TABLE users IS 'ユーザー情報';
COMMENT ON COLUMN users.email IS 'ログインに使用するメールアドレス';
```

また `.tbls.yml` でもコメントを追加・上書きできる。

```yaml
comments:
  - table: users
    tableComment: Users table
    columnComments:
      email: Email address as login id.
```

dbfolio ではコメントを重要情報として扱う。

特にテーブル詳細画面では、単なる ER 図ではなく「DB 仕様書」として読めることを重視する。

---

## 8. HTML の基本 UX

### Overview

最初のページで DB 全体を把握できるようにする。

例:

```text
Database: example

Tables       42
Views         5
Columns     386
Relations    67
Indexes      81
```

加えて以下を表示する。

- テーブル一覧
- コメント
- カラム数
- Relation 数
- Schema 単位の分類
- 全体 ER 図へのリンク

### Table detail

テーブル単位で詳細を確認できるページを中心にする。

例:

```text
users

Description
  ユーザー情報

Columns
────────────────────────────────────────────
Name       Type          Nullable   Key   Comment
id         bigint        NO         PK    ユーザーID
name       varchar(100)  NO               氏名
email      varchar(255)  NO               メールアドレス
region_id  bigint        YES        FK    地域ID

Indexes
...

Constraints
...

Parents
  regions

Children
  orders
  comments
  user_roles
```

### リンク

以下は HTML 内で相互リンクする。

- FK カラム → 親テーブル
- 親テーブル → 子テーブル
- ER 図上のテーブル → テーブル詳細
- テーブル一覧 → 詳細
- Relation 一覧 → 両端テーブル

---

## 9. ER 図

### 方針

ブラウザーで viz.js を実行して毎回レイアウトする方式は初期採用しない。

理由:

- WASM 読み込みが必要
- ブラウザーでレイアウト計算が発生
- 大規模スキーマで初期表示が重くなる可能性
- 静的ドキュメントとしては生成時に SVG を作る方が自然

そのため **生成時に SVG を生成して HTML から参照する**。

```text
schema.json
    ↓
Go で DOT を生成
    ↓
go-graphviz
    ↓
SVG
    ↓
HTML
```

---

## 10. Graphviz 実装方針

第一候補として **goccy/go-graphviz** を利用する。

Repository:

https://github.com/goccy/go-graphviz

特徴:

- Go から Graphviz を利用できる
- Graphviz の外部インストールが不要
- Graphviz を WASM 化した `graphviz.wasm` を内包
- DOT の encode / decode に対応
- `dot`, `neato`, `fdp`, `sfdp`, `circo` 等の layout を利用可能
- SVG / PNG / JPEG / DOT を生成可能
- CGO やシステム Graphviz への依存を避けられる

dbfolio では基本的に `dot` レイアウトを利用し、SVG を生成する。

概念的な処理:

```go
dotSource := buildDOT(schema)

g, err := graphviz.New(ctx)
if err != nil {
    return err
}
defer g.Close()

graph, err := graphviz.ParseBytes([]byte(dotSource))
if err != nil {
    return err
}
defer graph.Close()

var buf bytes.Buffer
if err := g.Render(ctx, graph, graphviz.SVG, &buf); err != nil {
    return err
}
```

### PoC での確認結果 (2026-10-04)

`poc/vertical-slice` ブランチで go-graphviz v0.2.10 (Graphviz 12.1.2 内包) を検証した。

- `graphviz.New` の初期化: 約 2ms
- 10 テーブル分のテーブル単位 ER 図 (distance=1) の SVG 生成: 合計約 23ms
- バイナリサイズ: 16.7MB (`-ldflags="-s -w"` で 11.8MB)
- PNG 出力 (Go 側のラスタライズ) は曲線が塗りつぶされ、日本語も描画されないため実用にならない。出力は SVG に限定する

### DOT の表現方針 (PoC で確認済み)

- テーブルは `shape=plaintext` + HTML ラベルの表形式で描き、`rankdir=LR` で子テーブルを左、親テーブルを右に配置する
- カラム名セルに west 用、型セルに east 用の port を置き、Relation は `子:east -> 親:west` で接続する (1 つの port だと線が型セルの文字に重なる)
- カーディナリティは crow's foot (`teetee` / `teeodot` / `crowodot` / `crowtee`) で、`dir=both` の両端に描く
- Virtual Relation は破線にする
- セルには背景色を指定する。背景のないセルはリンクのクリック対象にならず、文字の上でしか反応しない
- WASM 版 Graphviz はフォントファイルを持たず、日本語の文字幅を小さく見積もる。日本語を含むセルは dbfolio 側で幅を見積もって `width` を指定し、左寄せにする
- 複合 FK は先頭カラムの port にのみ接続している。構成カラムをどう表現するかは未決

### ライセンス注意

- `go-graphviz` 本体: MIT
- 内包される `graphviz.wasm`: Graphviz 由来の Eclipse Public License

リリース時は THIRD_PARTY_NOTICES 等で依存ライセンスを整理する。

---

## 11. elk-go について

代替候補として `d2lang/elk-go` もある。

https://github.com/d2lang/elk-go

特徴:

- Native Go の ELK 移植
- Layered layout
- ports
- compound graph
- self loop
- label
- orthogonal routing

特に「カラムの行そのものを port として FK 線を接続する」ような独自 ER 図では魅力がある。

ただし初期実装では、

```text
DOT → go-graphviz → SVG
```

の方が実装量が少なく、Graphviz による安定したレイアウト品質を得やすい。

よって初期判断:

1. **go-graphviz を採用**
2. ER 図の表現上の制約が明確になった場合に elk-go を再評価

---

## 12. 既存ツールとの位置付け

### tbls

```text
Database
   ↓
 schema analysis
   ↓
Markdown / JSON / ER / lint / diff
```

dbfolio は競合するのではなく補完する。

```text
tbls
 ↓
schema.json
 ↓
dbfolio
 ↓
Browsable static HTML
```

### Liam ERD

Liam ERD は tbls JSON を正式に入力としてサポートしている。

例:

```bash
npx @liam-hq/cli erd build \
  --format tbls \
  --input schema.json
```

Liam ERD は主にインタラクティブな ER 図に強い。

dbfolio は ER 図だけでなく、

- テーブル仕様
- カラム仕様
- Index
- Constraint
- コメント
- Parents / Children
- ナビゲーション

を含む「DB ドキュメント」として差別化する。

### SchemaSpy

DB に直接接続してメタデータを取得し、HTML ドキュメントを生成する Java 製ツール。

dbfolio は DB に接続せず、tbls の `schema.json` を入力とする点が異なる。

---

## 13. CLI 案

MVP ではシンプルにする。

```bash
dbfolio build schema.json
```

出力先指定:

```bash
dbfolio build schema.json -o ./docs
```

### 出力先

- 既定の出力先は `./dbfolio-docs` とする
- `docs` は既存の手書きドキュメントと衝突しやすいため既定値にしない
- tbls の既定出力先 `dbdoc` も、dbfolio の入力元 (`schema.json`) と混ざるため使わない
- `docs` など任意の場所へ出したい場合は `-o` で明示指定する

### 既存ディレクトリの扱い

- 出力先が存在しない、または空の場合はそのまま生成する
- dbfolio が生成したことを示すマーカーファイル `.dbfolio` がある場合は、中身を削除して再生成する (削除されたテーブルのページを残さないため)
  - マーカーは通常ファイルで、先頭が dbfolio の署名文字列のときだけ有効とする (シンボリックリンクや他ツールの同名ファイルでは削除しない)
  - 有効なマーカーがあっても、ファイルシステムのルート・ホームディレクトリ・`.git` を含むディレクトリは削除を拒否する (シンボリックリンクは解決して判定する)
  - マーカーは最後に削除し、途中で失敗しても `--force` なしで再実行できるようにする
- マーカーがない非空ディレクトリはエラーとする
- `--force` 指定時はマーカーがない非空ディレクトリにも書き込むが、既存ファイルは削除せず、マーカーも書かない (次回以降のビルドで利用者のファイルが削除対象にならないようにするため)
- 生成ファイルは既存のエントリを削除してから作成する (シンボリックリンクを辿って外部のファイルを書き換えないため)
- 入力の読み込みと検証は出力先に触れる前に行い、不正な入力で前回の出力を消さない

将来候補:

```bash
dbfolio serve schema.json
dbfolio diff before.json after.json
dbfolio version
```

ただし `serve` は MVP には不要。

静的 HTML が `file://` でも十分閲覧できることを優先する。

---

## 14. Go プロジェクト構成案

```text
dbfolio/
├── cmd/
│   └── dbfolio/
│       └── main.go
├── internal/
│   ├── input/
│   │   └── tbls/            ← tbls schema.json の読み込みと内部モデルへの変換
│   │       └── loader.go
│   ├── schema/              ← 入力形式に依存しない内部モデル
│   │   ├── model.go
│   │   └── relation.go
│   ├── diagram/
│   │   ├── dot.go
│   │   └── graphviz.go
│   ├── render/
│   │   ├── renderer.go      ← //go:embed を置く
│   │   ├── viewmodel.go
│   │   ├── templates/
│   │   │   ├── layout.html
│   │   │   ├── index.html
│   │   │   ├── table.html
│   │   │   └── relationships.html
│   │   └── assets/
│   │       ├── app.css
│   │       └── app.js
│   └── output/
│       └── writer.go
├── testdata/
│   └── schema.json
├── docs/
│   └── design/
│       ├── concept.md
│       └── roadmap.md
├── go.mod
└── README.md
```

リポジトリのドキュメントは `docs/` に置く。開発者向けの設計資料は `docs/design/` とする。

テンプレートと Asset は `embed` する。`//go:embed` は親ディレクトリを参照できないため、テンプレート・Asset は利用するパッケージ (`internal/render`) 配下に置く。

```go
//go:embed templates assets
var content embed.FS
```

これにより dbfolio 本体を単一バイナリで配布できる。

---

## 15. 内部モデル

tbls の JSON 構造を HTML テンプレートから直接参照しすぎない。

dbfolio 内部モデルへ変換する層を設ける。

理由:

- tbls JSON の変更を局所化
- View 用の逆引き情報を追加しやすい
- 将来 tbls 以外の入力形式を追加可能
- テストしやすい

内部モデルは `internal/schema/model.go` に定義する (Phase 2 で実装)。主な判断:

- テーブル種別は DB ごとに値が異なる (`BASE TABLE` / `table` / `FOREIGN TABLE` など) ため、元の文字列 (`Type`) と View かどうか (`View`、種別に `VIEW` を含むか) を分けて持つ
- 制約は `Kind` (PRIMARY KEY / FOREIGN KEY / UNIQUE / CHECK / OTHER) で分類し、元の種別文字列も保持する。カラム属性と重複する制約 (PostgreSQL 18 の NOT NULL 制約) は入力層で除外する
- PK / FK は Column にフラグとして持たせない。1 カラムが複数の FK や複合 FK に属しうるため、制約を正として持ち、カラム表の Key 表示は ViewModel で制約から逆引きする
- `Def` は入力が提供する場合のみ持つ (View の定義、MySQL では CREATE TABLE 文)

```go
type Table struct {
    Name        string
    Type        string
    View        bool
    Comment     string
    Columns     []*Column
    Indexes     []*Index
    Constraints []*Constraint
    Triggers    []*Trigger
    Def         string
}

type Constraint struct {
    Name              string
    Kind              ConstraintKind
    Type              string
    Def               string
    Columns           []string
    ReferencedTable   string   // FK のみ
    ReferencedColumns []string // FK のみ。ReferencedColumns[i] は Columns[i] が参照する
    Comment           string
}
```

さらに HTML 用 ViewModel を分ける。

```go
type TablePage struct {
    Table     Table
    Parents   []Relation
    Children  []Relation
    Diagram   string
}
```

描画は Plan (ページのファイル名の決定と重複検出) と Write (ファイルの書き出し) に分ける。入力に起因するエラーはすべて Plan までに検出し、出力先ディレクトリを削除・変更する前に失敗させる (不正な入力で前回の出力を消さないため)。

---

## 16. Relation の扱い

Relation を双方向に引ける Index (`schema.RelationIndex`) を作る (Phase 3 で実装)。

```text
Relation
  child = orders
  parent = users
```

から、

```text
orders:
  Parents = users

users:
  Children = orders
```

を構築する。

```go
type RelationIndex struct {
    parents  map[*Table][]*Relation
    children map[*Table][]*Relation
}
```

HTML 生成時に毎回全 Relation を走査しない設計にする。Index は描画の Plan で作る。

### 入力の解決と検証

- Relation の両端は、テーブル名の文字列ではなく `*schema.Table` へのポインターとして持つ。入力層で名前を解決する
- 両端テーブルやカラムが存在しない、子と親のカラム数が合わない、カラムがない Relation は、読み込み時にエラーとする。tbls 自身も生成時にこれらを検証するため、ここで失敗するのは schema.json が編集された場合などに限られる。黙ってスキップすると誤ったドキュメントになるため、エラーで知らせる
- 描画の Plan でも両端が Schema に含まれることを再確認する (入力層以外で組み立てた Schema でも、出力先の削除後に失敗しないため)

### 表示

- カラム表の Key 列の FK は、FK 制約ではなく Relation から導出する (FK 制約がない Virtual Relation も含めるため)。Relation ごとに FK バッジを出し、親テーブルへリンクする。1 カラムが複数の Relation に属する場合はバッジが複数並ぶ
- Virtual Relation は DB による整合性の保証がないため、通常の FK と区別して表示する (バッジは点線、表では "Virtual")。見た目だけに頼らず、スクリーンリーダー向けのテキストも付ける
- テーブル詳細の Parents / Children は相手側のテーブルのみ表示し、Relationships 一覧 (`relationships.html`) は両端を表示する
- カラムの対応は「子 → 親」で表記し、複合の場合は `(order_id, line_no) → (order_id, line_no)` とする
- カーディナリティは tbls の値を記号に変換し、`child : parent` の順で表記する (例: `0..* : 1`)

---

## 17. テーブル単位 ER 図

全体 ER 図だけでなく、テーブル中心 ER 図を生成する。

例:

```text
                    regions
                       ↑
                       │
roles ← user_roles ← users → orders → order_items
                       │
                       ↓
                    comments
```

`distance` を導入し、BFS で対象 Relation を抽出できるようにする。

概念 API:

```go
func RelatedTables(
    schema *Schema,
    table string,
    distance int,
) []string
```

MVP では distance=1 を基本とし、将来的に設定可能にする。

---

## 18. MVP

まず以下に限定する。進捗は [roadmap.md](roadmap.md) で管理する。

### 必須

- `schema.json` 読み込み
- Overview
- テーブル一覧
- テーブル詳細
- カラム一覧
- DB コメント
- PK
- FK
- Index
- Constraint
- Parents
- Children
- 全体 ER 図
- テーブル単位 ER 図
- テーブル間リンク
- テーブル検索
- CSS / JS / Template の embed
- 単一バイナリ配布

### MVP ではやらない

- DB への直接接続
- tbls と同等の DB driver 実装
- 既存の DB ドキュメントツールとの互換性
- Implied Relationship の独自推論
- Anomaly の高度な解析
- ブラウザー上での Graphviz layout
- React / Vue 等による SPA 化
- サーバー必須の UI

---

## 19. 将来機能

### Anomalies

候補:

- Primary Key がない
- Relation が一つもない orphan table
- FK カラムに適切な Index がない
- コメントがない
- 大量カラムを持つテーブル
- Duplicate Relation

ただし tbls 自体にも lint があるため、dbfolio で重複実装する価値を確認する。

### Diff

将来的に非常に有力。

```bash
dbfolio diff before.json after.json
```

例:

```text
Schema Changes

+ users.nickname varchar(100)
- users.legacy_code
~ orders.status varchar(10) → varchar(20)

+ FK orders.user_id → users.id
```

HTML でスキーマ変更を閲覧できるようにすると、dbfolio 独自の価値になる。

### 入力形式追加

将来的には以下も検討可能。

```text
tbls JSON
DBML
Prisma schema
SQL DDL
```

ただし MVP では tbls JSON に集中する。

---

## 20. 検索

MVP の検索はサーバーサイド機能を必要としない方法にする。

生成時に検索用インデックスを作る。

```text
assets/search-index.js
```

`file://` で開いた場合、ブラウザーは `fetch()` によるローカル JSON の読み込みをブロックする。そのため JSON ファイルではなく、インデックスをグローバル変数へ代入する JS ファイルとして出力し、`<script>` で読み込む。

```js
window.DBFOLIO_SEARCH_INDEX = [ /* ... */ ];
```

対象:

- テーブル名
- テーブルコメント
- カラム名
- カラムコメント

ブラウザー側で軽量な JavaScript により絞り込む。

大規模 Schema で問題になった場合に検索ライブラリ導入を検討する。

---

## 21. パフォーマンス方針

生成時に重い処理を寄せ、閲覧時は軽くする。

```text
Generation time:
  JSON parse
  Relation analysis
  DOT generation
  Graphviz layout
  SVG generation
  HTML generation

Browser:
  HTML display
  SVG display
  lightweight search
```

ER 図の Graphviz 計算をブラウザー側では行わない。

全体 ER 図が巨大になる場合は、以下を検討する。

- Schema 単位に分割
- Viewpoint
- Relation distance
- 全体図をオプション化
- テーブル中心図を基本 UI にする

---

## 22. テスト方針

### Unit Test

- schema.json parser
- internal model 変換
- Parent / Child Relation 構築
- 複合 FK
- Virtual Relation
- DOT 生成
- HTML ViewModel

### Golden Test

HTML や DOT の生成結果には Golden Test が有効。

HTML は生成サイト全体を `testdata/<case>/expected/` と比較する。期待値は `go test ./internal/render -update` で更新する。

```text
testdata/
├── basic/
│   ├── schema.json
│   ├── expected/        ← 生成サイト全体 (index.html, tables/, assets/)
│   └── expected.dot
├── composite-fk/
└── virtual-relation/
```

### Integration Test

```text
schema.json
 ↓
dbfolio build
 ↓
dbfolio-docs/
```

までを実行し、主要ファイルが生成されることを確認する。

---

## 23. 実装計画

Phase 構成・最初の PoC・進捗は [roadmap.md](roadmap.md) で管理する。

最初から多機能を目指さず、**「1 テーブルを見やすく、DB 仕様書として読める形で表示できるか」**を最初のゴールにする。

---

## 24. 現時点の主要技術判断

| 項目 | 判断 |
|---|---|
| プロジェクト名 | **dbfolio** |
| 言語 | Go |
| DB 直接接続 | MVP ではしない |
| 入力 | tbls `schema.json` |
| 出力 | Static HTML + SVG |
| 既定の出力先 | `./dbfolio-docs` (`-o` で変更可) |
| HTML | `html/template` |
| 閲覧環境 | `file://` で開いても全機能が動くこと |
| Assets | `embed` |
| 検索 | 生成時に `search-index.js` を出力し、ブラウザーで絞り込み |
| ER 表現 | DOT |
| ER renderer | **goccy/go-graphviz** |
| ER 図の埋め込み | inline SVG |
| Browser Graphviz | 使用しない |
| Graphviz 外部インストール | 不要 |
| tbls | DB schema extraction layer |
| CLI | cobra (位置引数の後ろのフラグも解析できるため) |
| SPA framework | MVP では使用しない |
| 配布 | 単一バイナリ (GitHub Release / Homebrew / Scoop) |
| リポジトリのドキュメント | `docs/` (設計資料は `docs/design/`) |

---

## 25. 設計上の注意事項

### tbls schema.json のバージョン互換性

tbls の JSON を外部契約として利用するものの、将来の変更には備える。

候補:

- 読み込み時に tbls version / schema version があれば記録
- 未知フィールドを許容
- 必須フィールドを最小限にする
- parser を `internal/input/tbls` のように隔離する

将来的には:

```text
internal/input/
├── tbls/
├── dbml/
└── prisma/
```

という構成にできる。

### HTML の URL

テーブル名に以下が入った場合を考慮する。

- schema prefix
- uppercase
- space
- 記号
- Unicode

PostgreSQL では tbls のテーブル名が `public.users` のようにスキーマ名付きになる。

ファイル名には安全な slug / encoded ID を使う。Phase 1 で以下のルールに決定した。

- 小文字化し、`[a-z0-9._-]` 以外を `_` に置換する。先頭の `.` も `_` にする (隠しファイル化を防ぐ)
- 変換で名前が変わった場合、Windows 予約名 (`con` / `nul` / `com0`〜`com9` など) の場合、空の場合は、元の名前の SHA-256 の先頭 8 桁を `-` 付きで付与する (例: `Users` → `users-xxxxxxxx`)
- 変換が不要な名前はそのまま使う (例: `public.users` → `public.users`)
- 100 文字を超える場合は切り詰めてハッシュを付与する
- 結果はテーブル名だけで決まる。ハッシュ衝突などは理論上ありうるため、複数ファイルを書く側で重複を検出する

表示名と URL を分離する。

### tbls schema.json の DB 依存の差異

- PostgreSQL 18 では NOT NULL も制約 (`type: "n"`) として出力される。カラム表に Nullable を表示するため、Constraints の表示からは除外する

DB ごとの差異は入力層 (`internal/input/tbls`) で吸収し、表示側に持ち込まない。

### SVG

ER 図は **inline SVG** で HTML に埋め込む (PoC で確定)。

- `<img>` で読み込んだ SVG は、仕様上リンクもツールチップも動作しない
- inline SVG ではノードのリンクが `file://` でも動作することを確認した
- 埋め込み時は Graphviz 出力の XML 宣言と DOCTYPE を取り除く
- 単体で開ける `diagrams/*.svg` も出力する (リンクは単体表示でも動作する)

---

## 26. 参考 URL

### tbls

https://github.com/k1LoW/tbls

特に確認すべき内容:

- JSON output
- schema.json datasource
- Relations
- Comments
- Virtual Relations
- JSON Schema

### go-graphviz

https://github.com/goccy/go-graphviz

確認点:

- Pure Go Library として利用可能
- Graphviz WASM 内包
- DOT
- SVG rendering
- License

### Liam ERD

https://liambx.com/docs/parser/supported-formats/tbls

tbls JSON を入力形式として利用している既存例。

### elk-go

https://github.com/d2lang/elk-go

将来、Graphviz より細かな port / routing 制御が必要になった場合の候補。
