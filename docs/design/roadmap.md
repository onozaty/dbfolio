# dbfolio ロードマップ

更新日: 2026-10-04

設計方針・技術判断は [concept.md](concept.md) を参照。本ファイルは「何をどの順で進めるか」と進捗を管理する。
細かいタスクは GitHub Issues で管理し、ここには Phase 単位の計画と完了条件のみを書く。

## 進捗サマリー

| Phase | 内容 | 状態 |
|---|---|---|
| 0 | 縦切り PoC | 未着手 |
| 1 | 最小 CLI | 未着手 |
| 2 | Table documentation | 未着手 |
| 3 | Relations | 未着手 |
| 4 | ER diagram | 未着手 |
| 5 | UX | 未着手 |
| 6 | Release | 未着手 |

---

## Phase 0: 縦切り PoC

ゴール: **1 テーブル (users) を HTML + ER 図で表示できること**。MVP 全体の技術リスクを潰す。

PoC のコードは捨てる前提で書いてよい。Phase 1 以降で構成を整え直す。

- [x] リポジトリ作成
- [ ] Go module 初期化
- [ ] tbls の sample `schema.json` を `testdata/` に追加
  - 複合 FK・Virtual Relation・コメント (日本語含む)・View を含むものにする
  - 再生成できるよう、元の DDL と `.tbls.yml` も一緒に置く (SQLite などで tbls から生成する)
- [ ] tbls JSON を読み込む最小 parser を実装
- [ ] 1 テーブルの HTML を生成
- [ ] Relation を抽出 (Parents / Children)
- [ ] 対象テーブル中心 (distance=1) の DOT を生成
- [ ] go-graphviz で SVG を生成
- [ ] HTML へ ER 図を表示

確認事項 (結果は concept.md または ADR に記録する):

- [ ] tbls JSON の扱いやすさ / Relation 情報の十分さ
- [ ] DOT で期待する ER 図を表現できるか
- [ ] go-graphviz の起動コスト / バイナリサイズ
- [ ] SVG 内リンクが機能するか (`file://` でも)
- [ ] inline SVG と `<img>` のどちらを採用するか
- [ ] HTML の方向性

完了条件: 上記確認事項に結論が出て、内部モデルと UI の方針が確定していること。

---

## Phase 1: 最小 CLI

```bash
dbfolio build schema.json [-o ./dbfolio-docs] [--force]
```

- [ ] CLI の骨格 (`build` サブコマンド、`version`)
- [ ] `schema.json` の読み込みとエラーハンドリング (ファイルなし / 不正 JSON / 未知フィールド許容)
- [ ] 動作確認する tbls のバージョンを決め、README に明記
- [ ] 出力ディレクトリ処理
  - [ ] 既定値 `./dbfolio-docs`
  - [ ] マーカーファイル `.dbfolio` による上書き判定
  - [ ] マーカーのない非空ディレクトリはエラー、`--force` で上書き
- [ ] テーブル名から安全なファイル名 (slug / encoded ID) を生成
- [ ] CI (go test / go vet / lint) の最小構成

完了条件: 任意の `schema.json` を渡して出力ディレクトリが安全に生成されること。

---

## Phase 2: Table documentation

- [ ] 内部モデル (`internal/schema`) と tbls 入力層 (`internal/input/tbls`) の分離
- [ ] Overview (DB 名、テーブル / View / カラム / Relation / Index 数)
- [ ] テーブル一覧
- [ ] テーブル詳細: Columns / Comments / Indexes / Constraints / Triggers
- [ ] PK / FK の表示
- [ ] テンプレート・Asset の `embed`
- [ ] Golden Test (HTML)

完了条件: 全テーブルの詳細ページが生成され、コメントを含めて仕様書として読めること。

---

## Phase 3: Relations

- [ ] RelationIndex (Parents / Children の双方向インデックス)
- [ ] テーブル詳細に Parents / Children を表示
- [ ] FK カラム → 親テーブルへのリンク
- [ ] 複合 FK
- [ ] Virtual Relation の表示 (通常の FK と区別できる表示)
- [ ] Relationships 一覧ページ

完了条件: テーブル間をリンクだけでたどれること。

---

## Phase 4: ER diagram

- [ ] DOT generator (Golden Test 付き)
- [ ] go-graphviz による SVG 出力
- [ ] テーブル単位 ER 図 (distance=1)
- [ ] 全体 ER 図
- [ ] ER 図上のテーブル → テーブル詳細へのリンク
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

PoC / 実装中に決める。決まったら concept.md を更新するか、`docs/design/adr/` に記録する。

- SVG の埋め込み方式 (inline / `<img>`)
- テーブル名からのファイル名生成ルール
- 全体 ER 図が巨大な場合の扱い (分割 / オプション化)
- CLI ライブラリの選定 (標準 `flag` / cobra 等)
