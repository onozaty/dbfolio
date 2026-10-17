# CLAUDE.md

## プロジェクト概要

dbfolio は、tbls が出力する `schema.json` から、ブラウザーで閲覧できる静的 HTML のデータベースドキュメントと ER 図 (SVG) を生成する Go 製 CLI ツール。

- 設計・技術判断: [docs/design/concept.md](docs/design/concept.md)
- 今後の作業 (機能の候補・未決事項): [GitHub Issues](https://github.com/onozaty/dbfolio/issues)
- リリース手順: [docs/release.md](docs/release.md)

作業前に concept.md と関連する Issue を確認し、方針に沿って進めること。

## 設計上の前提

- DB には接続しない。DB 解析は tbls に任せ、dbfolio は `schema.json` の閲覧用ドキュメント化に集中する
- 生成物は `file://` で開いても全機能が動くこと (`fetch()` でローカルファイルを読まない)
- ER 図は生成時に goccy/go-graphviz で SVG 化する。ブラウザーでレイアウト計算はしない
- テンプレート・Asset は `embed` し、単一バイナリで配布する
- CLI の既定の出力先は `./dbfolio-docs`。リポジトリの `docs/` は本リポジトリのドキュメント用

## ドキュメントの運用

- リポジトリのドキュメントは `docs/`、開発者向けの設計資料は `docs/design/` に置く
- concept.md には設計と判断のみを書き、今後の作業 (計画・要望・未決事項) は GitHub Issues で管理する (両方に同じ計画を書かない)
- Issue に対応する PR は、本文に `Closes #<番号>` を書いて Issue を閉じる
- PoC や実装で決まった技術判断は concept.md に反映する。比較検討の経緯や計測結果など、concept.md に書くと長くなる内容があるときだけ `docs/design/adr/` を作って記録する

## 開発環境

- Dev Container (Go 1.27) で開発する
- PostgreSQL 18 が `localhost:5432` で利用できる (接続情報は `PG*` 環境変数に設定済み)。tbls でサンプルの `schema.json` を生成する用途に使う (テスト用は `testdata/basic/`、デモサイト用は `demo/`。手順はそれぞれの README)

## ブランチ運用

- コードの変更はブランチを切って PR を出す (例: `feat/cli-build`)
- PoC は `poc/` ブランチで行い、main にはマージしない。結果は concept.md (必要なら ADR) に残す
- ドキュメントのみの小さな修正 (誤字や記述の更新など) は main に直接コミットしてよい

## コミット

- Conventional Commits 形式、件名は英語 (例: `feat: add search for tables and columns`)
- コミット時は `git-commit` スキルの手順に従う
