# basic サンプル

複合 PK / 複合 FK、Virtual Relation、日本語コメント、View を含むサンプルスキーマ。

`schema.json` は以下の手順で再生成する (Dev Container の PostgreSQL と tbls を使用)。

```bash
psql -f testdata/basic/schema.sql
tbls out -c testdata/basic/.tbls.yml -t json -o testdata/basic/schema.json
```
