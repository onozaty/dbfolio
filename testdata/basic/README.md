# basic サンプル

複合 PK / 複合 FK (単独 FK と重なるカラムを含む) / 複合 UNIQUE / 複合インデックス、Virtual Relation、自己参照 FK、トリガー (関数)、enum 型、日本語コメント、View を含むサンプルスキーマ。

`schema.json` は以下の手順で再生成する (Dev Container の PostgreSQL と tbls を使用)。

```bash
psql -f testdata/basic/schema.sql
tbls out -c testdata/basic/.tbls.yml -t json -o testdata/basic/schema.json
```

`expected/` は生成サイト、`expected-dot/` は ER 図の DOT の Golden ファイル。`schema.json` やテンプレートを変更したら以下で更新し、差分を確認する。

```bash
go test ./internal/render ./internal/diagram -update
```

`expected/` の HTML は Graphviz が出力した SVG (座標を含む) を埋め込んでいるため、go-graphviz を更新すると全体が変わる。DOT 自体の変化は `expected-dot/` の差分で確認する。
