# dbfolio プロジェクト設計メモ

更新日: 2026-10-17

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

静的 HTML。ER 図は生成時に SVG にして HTML に埋め込む (inline SVG)。

```text
dbfolio-docs/
├── .dbfolio              ← dbfolio が生成したことを示すマーカー (§13)
├── index.html            ← Tables (テーブル一覧)
├── relationships.html    ← 全体 ER 図 (inline SVG) + 関係の一覧
├── functions.html        ← 関数がある場合のみ
├── enums.html            ← enum 型がある場合のみ
├── tables/
│   ├── public.users.html ← テーブル詳細。テーブル単位 ER 図 (inline SVG) を含む
│   └── ...
└── assets/
    ├── app.css
    ├── app.js            ← ER 図のドラッグ移動
    ├── search.js         ← 検索
    └── search-index.js   ← 検索用インデックス (生成時に作る)
```

Web サーバーや JavaScript フレームワークを必須にせず、基本は静的ファイルだけで閲覧できる形を目指す。

テーブル一覧はトップページ (`index.html`) とし、独立した `tables.html` は作らない。ページ間のリンクは相対パスにする (`file://` で開けるようにするため)。

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

## 6. schema.json から利用している情報

tbls の JSON Schema (`spec/tbls.schema.json_schema.json`) と照らし合わせて、以下を利用している。

### Database / Schema

- DB 名、説明 (`desc`)
- 関数・ストアドプロシージャ (`functions`)
- enum 型 (`enums`)
- `driver.meta.search_paths` (enum 型の名前解決に使う)

利用していないもの: `driver` の DB 名とバージョン (表示しない。§8)、`viewpoints` (表示方法を検討中。[#19](https://github.com/onozaty/dbfolio/issues/19))、`labels` (表示や絞り込みは予定しない。ビューポイントの対象指定に使う場合は #19 で扱う)

### Table

- テーブル名
- コメント
- カラム
- Index
- Constraint
- Trigger
- View 等の種別情報
- 定義 (`def`。View のクエリなど)
- View の参照先テーブル (`referenced_tables`)

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

### ページ構成

メニューは Tables / Relationships とし、関数や enum がある場合は Functions / Enums を加える (Phase 5 で決定)。

| メニュー | ページ | 内容 |
|---|---|---|
| Tables | `index.html` | DB の説明 (あれば) とテーブル一覧 (名前 / 種別 / カラム数 / コメント) |
| Relationships | `relationships.html` | テーブル間の関係。全体 ER 図 (キーカラムのみ / 全カラム) と一覧をタブで切り替える |
| Functions | `functions.html` | 関数・ストアドプロシージャ (名前 / 種別 / 引数 / 戻り値の型)。ある場合のみメニューに出す |
| Enums | `enums.html` | enum 型 (名前 / 値)。ある場合のみメニューに出す |

- 当初の Overview ページは Tables ページにまとめた。最初にしたいのは目的のテーブルを探すことで、件数はその補足になるため
- 件数のサマリー (カード) は置かず、各ページの見出しの横に件数を添える (例: `Tables 9 tables, 1 view`、`Relationships 10 relationships`)。サマリーは下の一覧やメニューと重複し、場所を取るため
- 一時期、SchemaSpy にならって全カラムの一覧 (Columns) と全制約の一覧 (Constraints) のページを置いたが、削除した。カラムを探す用途は検索で満たせ、制約名 (エラーメッセージに出る) から探す用途も検索の対象に制約名とインデックス名を加えて満たせるため。寄せる対象の tbls の Markdown にもなく、大規模 DB では巨大なページになる
- 関係の一覧は Relationships に置く。テーブル間の関係は図で把握するのが自然で、一覧は図の補助になるため
- Virtual Relation は DB に存在しない関係で制約ではない。テーブル詳細の Constraints セクションは DB の制約のみ、Relationships (Parents / Children) は Virtual Relation を含む
- 現在のページはメニューに `aria-current="page"` で示す
- スマートフォン向けの専用レイアウトは作らない。DB の仕様書は主に PC で見るもので、中身の大半が横に長い表と ER 図のため作り込みの効果が小さい。狭い画面では、ヘッダーの折り返しと表・図の横スクロールで崩れないようにする (幅 400px 程度で確認済み)
- 関数と enum は tbls の README ではトップページに並ぶが、メニュー名 (Tables) と中身が合わなくなるため別ページにする。多くの DB は関数も enum も持たないため、ある場合だけメニューに出す
- 関数は tbls の schema.json に名前・種別・引数・戻り値の型しかない (定義やコメントは出力されない) ため、一覧のみで個別ページは作らない
- enum 型のカラムは、型名を Enums ページの該当行へのリンクにし、ツールチップに値を出す。PostgreSQL は search_path 上のスキーマを省いた型名 (`order_status`) を出すため、スキーマなしの型名は tbls の `driver.meta.search_paths` の順に解決する (DB と同じ解決順になり、別スキーマの同名の型と取り違えない)。search_paths がない場合は、スキーマを除いた名前が一意なときだけ照合する。配列 (`order_status[]`) も同じ enum を指すとみなし、引用符付きの型名 (`public."OrderStatus"`) は引用符を除いて照合する。MySQL の ENUM はカラムの型に値が含まれるため対象外
- DB の種類とバージョン (`driver.name` / `database_version`) は表示しない。tbls の Markdown も出しておらず、仕様書を読む上での必要性が低い一方、表示用に短くする処理が DB ごとに必要になるため。`driver.meta.search_paths` は enum の照合に使う
- ビューのテーブル詳細には参照先テーブル (Referenced Tables) を出す。スキーマ外のテーブル (`.tbls.yml` で除外したものなど) はリンクなしで出す

### Table detail

テーブル単位で詳細を確認できるページを中心にする。セクションの順と表の列は tbls が生成する Markdown に寄せる (Phase 5 で見直し)。

```text
users
  BASE TABLE
  ユーザー情報                      ← テーブルコメント
  ▶ Definition                     ← ビューの定義など (折りたたみ。入力にある場合のみ)

Columns
  Name       Type          Default  Nullable  Key  Comment
  id         bigint        nextval  NO        PK   ユーザーID
  region_id  bigint                 YES       FK   地域ID

Constraints  (Name / Type / Definition / Comment)
Indexes      (Name / Definition / Comment)
Triggers     (Name / Definition / Comment)

Relationships                      ← メニューの Relationships ページと同じく図 + 一覧
  (テーブル単位 ER 図)
  Parents / Children               ← Virtual Relation を含む関係
```

- Nullable は YES / NO で表記する (SQL 標準の information_schema や MySQL と同じ。tbls の true / false は JSON の真偽値をそのまま出したもので、仕様書として読む人には DB 側の表記が自然なため)
- Indexes / Constraints / Triggers の Comment 列は、コメントが 1 つもなければ出さない (tbls と同様)。コメントは DB (PostgreSQL の `COMMENT ON INDEX` / `COMMENT ON CONSTRAINT` / `COMMENT ON TRIGGER`、MySQL のインデックスの `COMMENT`) や `.tbls.yml` の `indexComments` / `constraintComments` / `triggerComments` で付けられるが、付けている DB は少ない。
- 関係は Relationships セクションにまとめ、ER 図で全体像を見せてから Parents / Children の一覧を続ける (tbls は末尾に ER 図だけを「Relations」として置くが、サイトのメニュー名と構成にそろえる)。Relation のないテーブルではセクションを出さない
- 親子関係は tbls のようにカラムごとの Children / Parents 列にはせず、Key 列の FK バッジと Parents / Children セクションで示す (子テーブルが多いと列が横に長くなるため)

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
- 各行の先頭のセルに west 用、末尾のセルに east 用の port を置き、Relation は `子:east -> 親:west` で接続する (1 つの port だと線が他のセルの文字に重なる)。セルの構成は「実装上の注意」を参照
- カーディナリティは crow's foot (`teetee` / `teeodot` / `crowodot` / `crowtee`) で描く。Phase 4 で子側の端だけに描くことにした (SchemaSpy と同様)。同じ親カラムに多くの線が集まると親側の記号が重なって潰れるため。親側のカーディナリティ (FK が NULL を許すか) はテーブル詳細と Relationships 一覧で確認できる
- Virtual Relation は破線にする
- セルには背景色を指定する。背景のないセルはリンクのクリック対象にならず、文字の上でしか反応しない
- 複合 FK は先頭カラム同士の port にのみ接続し、全カラムの対応は線のツールチップに出す (Phase 4 で決定。構成カラムごとに線を引くと本数が増えて読みにくくなるため)

### 実装上の注意 (Phase 4)

- DOT 生成 (`internal/diagram/dot.go`) は go-graphviz に依存しない純粋な関数にし、Golden Test (`testdata/basic/expected-dot/`) で検証する
- ノード ID はテーブルの並び順 (`t0`, `t1`, ...)、port はカラムの並び順 (`w0` / `e0`, ...) にする。名前に記号を含んでも DOT が壊れないようにするため
- XML 1.0 で使えない文字 (タブ・改行以外の制御文字など) は U+FFFD に置き換える。HTML ラベルは XML としてパースされ、これらの文字があるとラベル全体が警告だけで失われる (エラーにならない)
- DOT の文字列は `\` と `"` をエスケープする (`tooltip` や `URL` は `\N` などを置換するため)。HTML ラベルは `& < > "` をエスケープする
- 空の `<font></font>` は HTML ラベルとして不正で、ラベル全体 (と port) が失われる。空文字のときは要素を出さない
- go-graphviz の `Close` は、コンテキスト全体で発生したエラーと警告の件数をエラーとして返す (非 ASCII 文字の幅の警告を含む)。どの図で起きたかは分からず、ラベルの構文エラーのように警告にしかならない問題もあるため、`Close` の結果は無視し、DOT 側で構文エラーを起こさないようにすることを前提にする
- 描画の Plan では、Relation のカラムの存在と数の一致も再確認する (ER 図の線の接続先に使うため)。`Write` は `RenderDiagrams` を呼ばずに実行するとエラーにする
- SVG の生成は出力先の準備 (`output.Prepare`) より前に行い、Graphviz の失敗で前回の出力を消さないようにする
- テーブル単位 ER 図は distance=1 の範囲のテーブルと、その間の Relation をすべて描く。Relation のないテーブルには ER 図を出さない
- 全体 ER 図は「キーカラムのみ」と「全カラム」の 2 種類を生成し、ページ上のタブで切り替える (既定はキーカラムのみ。SchemaSpy の Compact / Large と同じ考え方)。キーカラムのみでは PK と Relation のカラムだけを型なしで表示し、残りは「… N more columns」とまとめる。タブはラジオボタンと CSS で実装し、JavaScript なしでも動く。同じページに 2 つの SVG を置くため、要素の id に接頭辞を付けて重複を避ける
- テーブル単位 ER 図は全カラムと型を表示する
- PK / FK は行の右端 (型の後ろ) に "PK" / "FK" の文字ラベルで示す (Mermaid の erDiagram や Visio と同様。色は型と同じグレーにする。種類は文字で分かり、色付きのラベルだけが浮いて見えるため。ラベルはカラム名より小さい 9pt の通常の太さにする。Graphviz は文字をサイズごとに縦中央へ配置し、太字は背を高く見積もるため、小さい文字や太字は隣の文字より上に寄る。ラベルのセルは `valign="bottom"`、`cellpadding="6"` にしてカラム名とベースラインをそろえる)。PK かつ FK のカラムは PK だけにする (幅を取りすぎないため)。どのカラムもキーでないテーブル (View など) ではラベルの列を出さない。行の先頭のセルを west 側、末尾のセルを east 側の port にする。Relation の子側のカラムには必ず FK ラベルが付くため、線が出る行は常に 2 セル以上になる (1 セルになるのは、キーカラムのみの表示で PK のない親専用のテーブルだけで、線を受ける west 側しか使わない)
- 鍵アイコン (SchemaSpy 風) も試したが採用しなかった。Graphviz の HTML ラベルにはベクター図形を置けないため SVG の後処理で目印の文字を図形に置き換える必要があり、さらに SVG のテキスト先頭の空白はブラウザーで詰められるなど、位置合わせの調整が続いたため。文字ラベルなら Graphviz の描画だけで完結する
- 列の幅は dbfolio 側で計算し、すべての行に同じ幅を指定する。見出しが列の合計より広いと、Graphviz は余った幅をすべての列に分配するため。余りはカラム名の列に寄せる
- FK は Schema 全体の Relation から判定する (Virtual Relation を含め、親テーブルが図の外にあっても付く)
- テーブルコメントは図に表示せず、ノードのツールチップでのみ見せる (コメントのあるテーブルとないテーブルで見た目がそろわないため)
- 見出しのテーブル名は左寄せにする。ビューなど普通のテーブル以外は、見出しの右端に種別を `[VIEW]` のように小さく添える (SchemaSpy と同様。例: `[MATERIALIZED VIEW]`、`[FOREIGN TABLE]`)。普通のテーブル (`BASE TABLE` / `table` など) には付けない。種別の値は DB ごとに違い、図の大半を占めるテーブルに付けるとノイズになるため。色や破線は使わない (色は中心テーブルの強調、破線は Virtual Relation で使っているため)。名前と種別は見出しのセル内の罫線のない表に置き、Graphviz がセル幅まで引き伸ばすことを利用して左右の端に寄せる
- SVG のグラフ全体の `<title>` (グラフ名) は削除する。ブラウザーが図の余白にツールチップとして表示してしまうため
- 図の間隔は `nodesep=0.3`、`ranksep=0.8`、フォント 11pt、`cellpadding=5` とする (当初より詰めたが、詰めすぎると読みにくくなるため)
- WASM 版 Graphviz はフォントファイルを持たず、文字幅を組み込みの値で見積もる。ASCII はほぼ正確だが、全角文字は大きく狭い。また列幅をそろえるためにも (前項)、すべてのセルに dbfolio 側で見積もった幅 (ASCII 0.5em、全角 1.05em + 余白) を指定する。ASCII は Graphviz の見積もり (約 0.47em) とブラウザーでの実幅がほぼ一致するので、多めに見積もると余白が空きすぎる

### ライセンス注意

- `go-graphviz` 本体: MIT
- 内包される `graphviz.wasm`: Graphviz 12.1.2 (EPL-1.0)、Expat 2.6.3 (MIT)、wasi-libc (MIT を選択。一部 musl / cloudlibc などのライセンス) から作られている

依存ライセンスは THIRD_PARTY_NOTICES (リポジトリのルート) にまとめる (Phase 6)。

- `scripts/notices` で生成する。リリース対象の OS (linux / darwin / windows) ごとに実際にリンクされる Go モジュールを `go list -deps` で集め、モジュールキャッシュからライセンスファイルを取り出す。Go の標準ライブラリも含める
- `graphviz.wasm` の中身は Go モジュールからは分からないため、ライセンス文と入手先を `third_party/graphviz-wasm/` に置いて追加する。go-graphviz を更新したら、内包する Graphviz などのバージョンを確認してここも更新する
- EPL-1.0 は、オブジェクトコードとして別のライセンスで配布する場合に、ソースコードの入手方法を示すことを求める。Graphviz のソースの入手先 (12.1.2 のタグとリリースアーカイブ) を記載する
- golang/freetype は FreeType License と GPLv2 のデュアルライセンスのため、FreeType License (ftl.txt) を選ぶ
- CI で再生成して差分がないことを確認し、依存の更新時の更新漏れを防ぐ
- ライセンスを表示するコマンド (`dbfolio licenses` など) は設けず、リリースのアーカイブに同梱する

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

v1.0.0 時点では Graphviz で必要な表現ができているため、elk-go の再評価は予定しない ([#26](https://github.com/onozaty/dbfolio/issues/26) を not planned として閉じた)。

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

コマンドは `build` と `version` (cobra の `completion` / `help` も使える)。

将来候補:

```bash
dbfolio serve schema.json
```

ただし `serve` は MVP には不要。スキーマの差分表示 (`dbfolio diff`) は予定しない (§19)。

静的 HTML が `file://` でも十分閲覧できることを優先する。

---

## 14. Go プロジェクト構成

```text
dbfolio/
├── cmd/dbfolio/main.go          ← エントリーポイント (バージョンは -ldflags で埋め込む)
├── internal/
│   ├── cli/                     ← cobra のコマンド (build / version)
│   ├── input/tbls/              ← tbls schema.json の読み込み・検証と内部モデルへの変換
│   ├── schema/                  ← 入力形式に依存しない内部モデル、RelationIndex
│   ├── diagram/                 ← DOT 生成 (dot.go) と go-graphviz による SVG 化 (graphviz.go)
│   ├── render/                  ← ページの生成
│   │   ├── renderer.go          ← //go:embed、Plan / RenderDiagrams / Write
│   │   ├── viewmodel.go         ← テンプレート用のデータ
│   │   ├── search.go            ← 検索インデックス
│   │   ├── enums.go             ← enum 型の名前解決
│   │   ├── templates/           ← layout / partials / index / table / relationships / functions / enums
│   │   ├── assets/              ← app.css / app.js / search.js
│   │   └── testdata/            ← search.js の node テスト
│   └── output/                  ← 出力先ディレクトリの準備 (dir.go) とファイル名 (filename.go)
├── scripts/notices/             ← THIRD_PARTY_NOTICES の生成
├── third_party/graphviz-wasm/   ← graphviz.wasm に含まれるもののライセンス
├── testdata/basic/              ← サンプルスキーマと Golden ファイル
├── demo/                        ← デモサイト用の英語のサンプルスキーマ (GitHub Pages で公開)
├── docs/
│   ├── release.md               ← リリース手順
│   └── design/                  ← concept.md (設計と判断)
├── .goreleaser.yaml
├── THIRD_PARTY_NOTICES
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
type Schema struct {
    Name      string
    Desc      string
    Tables    []*Table
    Relations []*Relation
    Functions []*Function
    Enums     []*Enum
    Driver    *Driver   // search_paths のみ
}

type Table struct {
    Name             string
    Type             string
    View             bool
    Comment          string
    Columns          []*Column
    Indexes          []*Index
    Constraints      []*Constraint
    Triggers         []*Trigger
    Def              string
    ReferencedTables []string // View の参照先。スキーマ外のテーブルもありうるため名前で持つ
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

さらに HTML 用 ViewModel (`internal/render/viewmodel.go`) を分ける。テーブル詳細の ViewModel は、カラム行 (Key バッジ付き)、Parents / Children、テーブル単位 ER 図の SVG、各セクションのコメント列の有無などを持つ。

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

`distance` を導入し、BFS で対象テーブルを抽出する (`RelationIndex.RelatedTables(t *Table, distance int) []*Table`。親・子の両方向にたどり、結果は Schema のテーブル順)。図にはそのテーブル間の Relation をすべて描く。

MVP では distance=1 とし、将来的に設定可能にする ([#24](https://github.com/onozaty/dbfolio/issues/24))。

---

## 18. MVP

まず以下に限定する。以下はすべて実装済みで、v1.0.0 としてリリースした。

### 必須

- `schema.json` 読み込み
- テーブル一覧
- テーブル詳細
- カラム一覧 (テーブル詳細の Columns)
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

tbls 自体の lint (`tbls lint`) がより細かく担っているため、予定しない ([#22](https://github.com/onozaty/dbfolio/issues/22) を not planned として閉じた)。

### Diff

当初は有力な候補と考えていたが、v1.0.0 のあとに予定しないことにした ([#21](https://github.com/onozaty/dbfolio/issues/21) を not planned として閉じた)。dbfolio はスキーマを閲覧するためのドキュメントに集中する。以下は当初の案。

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

予定しない ([#25](https://github.com/onozaty/dbfolio/issues/25) を not planned として閉じた)。DB やスキーマ形式の解析は tbls に任せ、dbfolio は tbls の schema.json に集中する。他の形式は tbls などを経由して schema.json にできる。

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
- 制約名・インデックス名 (エラーメッセージや実行計画に出る名前から、テーブルを探せるようにする)

ブラウザー側で軽量な JavaScript (`assets/search.js`、依存ライブラリなし) により絞り込む。

Phase 5 で以下のように実装した。

- 全ページのヘッダーに検索ボックスを置く。JavaScript が動くときだけ表示する。表示の切り替えは `visibility` で行い、最初から場所を確保しておく (`display` で切り替えると、表示された瞬間にヘッダーの高さが変わってちらつくため)
- 部分一致 (大文字小文字を区別しない)。空白区切りは AND。カラムは `テーブル名.カラム名` でも一致する
- 並び順: テーブル名 → カラム名 → 制約名・インデックス名 → コメントの順。名前はそれぞれ前方一致を部分一致より上にする。同順位は名前の短い順。前方一致はスキーマ名を除いた部分 (`public.users` の `users`) でも判定する。最大 20 件
- 候補を選ぶとテーブル詳細へ移動する。カラム・制約・インデックスの場合はその行 (`#column-<番号>` / `#constraint-<番号>` / `#index-<番号>`) へ移動して強調表示する。名前には任意の文字が入りうるため、id には位置を使う
- ↑↓ / Enter / Esc で操作でき、`/` で検索ボックスへ移動する (GitHub と同様。検索ボックスの右端に `/` のキー表示を出す)。IME の変換中のキー (変換確定の Enter など) は操作として扱わない
- combobox / listbox の ARIA (`aria-activedescendant`、`aria-selected`) を付け、件数は `aria-live` の領域で読み上げる
- 索引は読み込み時に一度だけ小文字化し、入力ごとには比較だけを行う。2,000 テーブル × 25 カラム (52,000 件) で 1 回の検索は最大約 15ms (node、ほぼ全件が一致する短い語のとき)
- インデックスのリンクはルートからの相対パスで持ち、`<html data-root>` で各ページの階層に合わせる
- インデックスは `json.Marshal` で出力する (`<` `>` `&` と U+2028 / U+2029 がエスケープされ、コメントに `</script>` などがあっても壊れない)
- 候補の表示は `textContent` で行う (名前やコメントは DB 由来の任意の文字列のため)
- 検索ロジックは関数として切り出し、node でテストする (`internal/render/testdata/search_test.js`。node がない環境ではスキップ)

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

Phase 4 での計測 (合成スキーマ。各テーブル 4〜15 カラム、0〜3 個の FK。全体 ER 図はキーカラムのみ / 全カラムの 2 種類を生成):

| テーブル数 | Relation 数 | 生成時間 | 全体 ER 図のページ | 出力合計 | 全体 ER 図 (キーのみ) | 全体 ER 図 (全カラム) |
|---|---|---|---|---|---|---|
| 100 | 136 | 0.8 秒 | 1.2MB | 4.6MB | 1576 × 4092pt | 1895 × 8679pt |
| 300 | 426 | 2.6 秒 | 3.8MB | 15MB | 2220 × 11484pt | 2671 × 27344pt |
| 500 | 724 | 4.4 秒 | 6.6MB | 25MB | 2542 × 16699pt | 3059 × 28999pt |

生成時間は問題ない。全体 ER 図は数百テーブルで縦に長くなり、図全体を見渡す用途には向かない。テーブル単位 ER 図を基本の導線とする。

全体 ER 図が巨大になる場合は、以下を検討する ([#20](https://github.com/onozaty/dbfolio/issues/20))。

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

HTML は生成サイト全体を `testdata/basic/expected/` と、ER 図の DOT は `testdata/basic/expected-dot/` と比較する。期待値は `go test ./internal/render ./internal/diagram -update` で更新する。

```text
testdata/basic/
├── schema.sql / .tbls.yml   ← サンプルスキーマ (複合 PK / FK、Virtual Relation、自己参照、enum、View など)
├── schema.json              ← tbls で生成
├── expected/                ← 生成サイト全体
└── expected-dot/            ← 全体 ER 図 (キーのみ / 全カラム) とテーブル単位 ER 図の DOT
```

ケースは basic 1 つにまとめ、境界条件 (空のスキーマ、不正な Relation、エスケープなど) はユニットテストで確認する。

### JavaScript のテスト

検索ロジック (`assets/search.js` の `search` / `prepare`) は node でテストする (`internal/render/testdata/search_test.js` を Go のテストから実行。node がなければスキップ)。DOM 操作はブラウザーで確認する。

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

最初から多機能を目指さず、**「1 テーブルを見やすく、DB 仕様書として読める形で表示できるか」**を最初のゴールにした。

PoC (Phase 0) から Release (Phase 6) までの段階に分けて進め、v1.0.0 (2026-10-17) をリリースした。各段階の計画とチェックリストは `docs/design/roadmap.md` に置いていたが、リリース後に廃止した (git の履歴に残っている)。今後の作業は [GitHub Issues](https://github.com/onozaty/dbfolio/issues) で管理する。

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
| 配布 | 単一バイナリ (GitHub Release / Homebrew / Scoop)。GoReleaser でビルドし、CGO なしで linux / darwin / windows × amd64 / arm64 を作る |
| リポジトリのドキュメント | `docs/` (設計資料は `docs/design/`) |

---

## 25. 設計上の注意事項

### tbls schema.json のバージョン互換性

tbls の JSON を外部契約として利用するものの、将来の変更には備える。以下を実装している。

- 未知フィールドを許容する (新しい tbls の出力も読める)
- 必須フィールドは `tables` と各テーブルの `name` のみとし、その他の不足は空として扱う (Relation は両端の解決に必要な項目を検証する)
- parser を `internal/input/tbls` に隔離する
- 動作確認した tbls のバージョンを README に明記する

schema.json には tbls のバージョンが含まれないため、読み込み時の記録はしていない。

入力層を分離しているので、他の形式を読む構成 (`internal/input/dbml` など) にもできるが、入力形式の追加は予定しない (§19)。

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
- 図は原寸で表示し、枠からはみ出す部分はスクロールで見る。Relationships ページはページ自体をスクロールさせず、図の枠が画面の残りの高さに収まるようにする (スクロールバーが二重にならないように)。テーブル詳細の ER 図は長いページの途中にあるため、枠の高さを画面の 80% までにする。本文の幅は制限せず、どのページも画面幅いっぱいに使う (横に長い表や図を広い画面で見やすくするため)
- マウスのドラッグで図を移動 (パン) できるようにする (`assets/app.js`、依存ライブラリなし)。動かしたドラッグの直後のクリックは無視し、単純なクリックでのリンク遷移と両立させる
- 図の中のリンクにはホバー時の下線を付けない (ノード内の文字すべてに下線が付くため)
- ダークモード (`prefers-color-scheme: dark`) では、ページの CSS で SVG の色を置き換える (inline SVG なので CSS が Graphviz の属性の色より優先される)。対象は dbfolio が Graphviz に指定した色 (`white` / `black` / `#666666` / 見出しと中心テーブルの色) と色指定のない文字だけで、属性の値で選ぶ。SVG に出る色がすべて置き換えの対象になっていることはテストで確認する (DOT に色を足したときの漏れを防ぐ)。ライトモードの見た目は変えない
- ページは `color-scheme: light dark` を指定し、検索ボックスやスクロールバーなどのブラウザー部品もテーマに合わせる。手動の切り替え (ライト / ダーク / OS に従う) は設けない
- 単体の SVG ファイル (`diagrams/*.svg`) は出力しない (Phase 4 で決定)。単体ファイルと埋め込みとでリンクの相対パスが異なり、全体図は 2 回レンダリングが必要になるため。用途が出てきたら再検討する

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
