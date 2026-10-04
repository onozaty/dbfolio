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
├── index.html
├── tables.html
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
- dbfolio が生成したことを示すマーカーファイル (例: `.dbfolio`) がある場合は上書きする
- マーカーがない非空ディレクトリはエラーとし、`--force` 指定時のみ上書きする
- ディレクトリごと削除して作り直す動作は既定では行わない

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
│   │   │   ├── index.html
│   │   │   ├── tables.html
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

概念モデル:

```go
type Schema struct {
    Name      string
    Tables    []Table
    Relations []Relation
}

type Table struct {
    Name        string
    Comment     string
    Columns     []Column
    Indexes     []Index
    Constraints []Constraint
}

type Relation struct {
    Table             string
    Columns           []string
    ParentTable       string
    ParentColumns     []string
    Cardinality       string
    ParentCardinality string
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

---

## 16. Relation の扱い

ロード時に Relation を双方向に引ける Index を作る。

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

例:

```go
type RelationIndex struct {
    Parents  map[string][]Relation
    Children map[string][]Relation
}
```

HTML 生成時に毎回全 Relation を走査しない設計にする。

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

```text
testdata/
├── basic/
│   ├── schema.json
│   ├── expected.dot
│   └── expected.html
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
| Browser Graphviz | 使用しない |
| Graphviz 外部インストール | 不要 |
| tbls | DB schema extraction layer |
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

ファイル名には安全な slug / encoded ID を使う。

表示名と URL を分離する。

### SVG

Graphviz の SVG をそのまま `<img>` として扱うか、inline SVG にするかは PoC で比較する。

inline SVG の方が:

- Table node から HTML へリンク
- hover
- CSS customization

を行いやすい可能性がある。

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
