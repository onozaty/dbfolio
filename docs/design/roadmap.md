# dbfolio ロードマップ

更新日: 2026-10-14

設計方針・技術判断は [concept.md](concept.md) を参照。本ファイルは「何をどの順で進めるか」と進捗を管理する。
細かいタスクは GitHub Issues で管理し、ここには Phase 単位の計画と完了条件のみを書く。

## 進捗サマリー

| Phase | 内容 | 状態 |
|---|---|---|
| 0 | 縦切り PoC | 完了 (2026-10-04) |
| 1 | 最小 CLI | 完了 (2026-10-04) |
| 2 | Table documentation | 完了 (2026-10-14) |
| 3 | Relations | 未着手 |
| 4 | ER diagram | 未着手 |
| 5 | UX | 未着手 |
| 6 | Release | 未着手 |

---

## Phase 0: 縦切り PoC

ゴール: **1 テーブル (users) を HTML + ER 図で表示できること**。MVP 全体の技術リスクを潰す。

PoC のコードは捨てる前提で書いてよい。Phase 1 以降で構成を整え直す。

コードは `poc/vertical-slice` ブランチに残している (main にはマージしない)。結果は concept.md に反映済み。

- [x] リポジトリ作成
- [x] Go module 初期化
- [x] tbls の sample `schema.json` を `testdata/` に追加
  - 複合 FK・Virtual Relation・コメント (日本語含む)・View を含むものにする
  - 再生成できるよう、元の DDL と `.tbls.yml` も一緒に置く (Dev Container の PostgreSQL に DDL を流し、tbls で生成する)
- [x] tbls JSON を読み込む最小 parser を実装
- [x] 1 テーブルの HTML を生成
- [x] Relation を抽出 (Parents / Children)
- [x] 対象テーブル中心 (distance=1) の DOT を生成
- [x] go-graphviz で SVG を生成
- [x] HTML へ ER 図を表示

確認事項 (結果は concept.md に反映する。比較の経緯や計測結果は必要に応じて ADR に記録する):

- [x] tbls JSON の扱いやすさ / Relation 情報の十分さ
- [x] DOT で期待する ER 図を表現できるか
- [x] go-graphviz の起動コスト / バイナリサイズ
- [x] SVG 内リンクが機能するか (`file://` でも)
- [x] inline SVG と `<img>` のどちらを採用するか
- [x] HTML の方向性

完了条件: 上記確認事項に結論が出て、内部モデルと UI の方針が確定していること。

---

## Phase 1: 最小 CLI

```bash
dbfolio build schema.json [-o ./dbfolio-docs] [--force]
```

- [x] CLI の骨格 (`build` サブコマンド、`version`)
- [x] `schema.json` の読み込みとエラーハンドリング (ファイルなし / 不正 JSON / 未知フィールド許容)
- [x] 動作確認する tbls のバージョンを決め、README に明記
- [x] 出力ディレクトリ処理
  - [x] 既定値 `./dbfolio-docs`
  - [x] マーカーファイル `.dbfolio` による上書き判定
  - [x] マーカーのない非空ディレクトリはエラー、`--force` で上書き
- [x] テーブル名から安全なファイル名 (slug / encoded ID) を生成
- [x] CI (go test / go vet / lint) の最小構成

完了条件: 任意の `schema.json` を渡して出力ディレクトリが安全に生成されること。

出力は Overview の仮ページ (`index.html`、テーブル一覧のみ) とマーカーのみ。テンプレートの `embed` とテーブル詳細ページは Phase 2 で行う。

---

## Phase 2: Table documentation

- [x] 内部モデル (`internal/schema`) と tbls 入力層 (`internal/input/tbls`) の分離
- [x] Overview (DB 名、テーブル / View / カラム / Relation / Index 数)
- [x] テーブル一覧
- [x] テーブル詳細: Columns / Comments / Indexes / Constraints / Triggers
- [x] PK / FK の表示
- [x] テンプレート・Asset の `embed`
- [x] Golden Test (HTML)

完了条件: 全テーブルの詳細ページが生成され、コメントを含めて仕様書として読めること。

テーブル一覧は Overview (`index.html`) に含めた。Key 列の FK 表示は FK 制約のみが対象で、Virtual Relation は Phase 3 で扱う。

---

## Phase 3: Relations

- [ ] RelationIndex (Parents / Children の双方向インデックス)
- [ ] テーブル詳細に Parents / Children を表示
- [ ] FK カラム → 親テーブルへのリンク
- [ ] 複合 FK
- [ ] Virtual Relation の表示 (通常の FK と区別できる表示)
- [ ] Relationships 一覧ページ
- [ ] Relation の両端テーブルが存在しない場合の扱い (`.tbls.yml` の Virtual Relation の typo など)。出力先に触れる前に検出する

完了条件: テーブル間をリンクだけでたどれること。

---

## Phase 4: ER diagram

- [ ] DOT generator (Golden Test 付き)
- [ ] go-graphviz による SVG 出力
- [ ] テーブル単位 ER 図 (distance=1)
- [ ] 全体 ER 図
- [ ] ER 図上のテーブル → テーブル詳細へのリンク
- [ ] 複合 FK の図上の表現
- [ ] 大規模スキーマでの生成時間・図の可読性の確認

完了条件: 全体図とテーブル中心図が生成され、図からページへ遷移できること。

---

## Phase 5: UX

- [ ] 共通ナビゲーション
- [ ] テーブル検索 (`assets/search-index.js` + 軽量 JS、`file://` で動作すること)
- [ ] Responsive layout
- [ ] Dark mode (必要なら)
- [ ] 大規模 DB 向け表示改善 (Schema 単位の分類など)

完了条件: 中〜大規模スキーマでも目的のテーブルにすぐたどり着けること。

---

## Phase 6: Release

- [ ] THIRD_PARTY_NOTICES (go-graphviz: MIT、graphviz.wasm: EPL など依存ライセンスの整理)
- [ ] GoReleaser
- [ ] GitHub Release
- [ ] Homebrew Tap (macOS / Linux)
- [ ] Scoop bucket (Windows)
- [ ] README (使い方、tbls との連携手順)
- [ ] デモサイト (生成サンプルを GitHub Pages で公開するか検討)

完了条件: GitHub Release / Homebrew / Scoop のいずれからでもインストールでき、`dbfolio build` が使えること。

---

## MVP 後の候補

優先度は MVP 完了時に見直す。

- `dbfolio diff before.json after.json` (スキーマ差分の HTML 表示)
- Anomalies (tbls lint との重複を確認してから)
- ER 図の distance 設定
- 入力形式の追加 (DBML / Prisma / SQL DDL)
- elk-go の再評価 (Graphviz で表現しきれない場合)

## 未決事項

PoC / 実装中に決める。決まったら concept.md を更新する。比較検討の経緯や計測結果など、concept.md に書くと長くなる内容があるときだけ `docs/design/adr/` に記録する。

- 複合 FK を ER 図でどう表現するか
- 全体 ER 図が巨大な場合の扱い (分割 / オプション化)
