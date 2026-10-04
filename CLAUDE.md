# CLAUDE.md

## プロジェクト概要

dbfolio は、tbls が出力する `schema.json` から、ブラウザーで閲覧できる静的 HTML のデータベースドキュメントと ER 図 (SVG) を生成する Go 製 CLI ツール。

- 設計・技術判断: [docs/design/concept.md](docs/design/concept.md)
- 計画・進捗: [docs/design/roadmap.md](docs/design/roadmap.md)

作業前に上記 2 ファイルを確認し、現在の Phase と方針に沿って進めること。

## 設計上の前提

- DB には接続しない。DB 解析は tbls に任せ、dbfolio は `schema.json` の閲覧用ドキュメント化に集中する
- 生成物は `file://` で開いても全機能が動くこと (`fetch()` でローカルファイルを読まない)
- ER 図は生成時に goccy/go-graphviz で SVG 化する。ブラウザーでレイアウト計算はしない
- テンプレート・Asset は `embed` し、単一バイナリで配布する
- CLI の既定の出力先は `./dbfolio-docs`。リポジトリの `docs/` は本リポジトリのドキュメント用

## ドキュメントの運用

- リポジトリのドキュメントは `docs/`、開発者向けの設計資料は `docs/design/` に置く
- concept.md には設計と判断のみを書き、計画と進捗は roadmap.md に一本化する (両方に同じ計画を書かない)
- タスクを完了したら roadmap.md のチェックボックスと進捗サマリーを更新する
- PoC や実装で決まった技術判断は concept.md に反映するか、`docs/design/adr/` に記録する

## 開発環境

- Dev Container (Go 1.27) で開発する
- PostgreSQL 18 が `localhost:5432` で利用できる (接続情報は `PG*` 環境変数に設定済み)。tbls でサンプルの `schema.json` を生成する用途に使う

## コミット

- Conventional Commits 形式、件名は英語 (例: `docs: add design concept and roadmap`)
- コミット時は `git-commit` スキルの手順に従う
