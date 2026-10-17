# Demo schema

The schema behind the [dbfolio demo site](https://onozaty.github.io/dbfolio/): a small online shop on PostgreSQL, with English comments.

It covers what dbfolio shows: two schemas (`public` and `billing`), composite primary and foreign keys, a self-referencing table, composite unique constraints, enum types, a view and a materialized view, a trigger with its function, and virtual relations defined in `.tbls.yml`.

The site is published by `.github/workflows/pages.yml` when a release is published (or when the workflow is run manually).

## Regenerate schema.json

With the PostgreSQL and tbls of the dev container:

```bash
psql -c "DROP DATABASE IF EXISTS dbfolio_demo" -c "CREATE DATABASE dbfolio_demo"
psql -d dbfolio_demo -f demo/schema.sql
tbls out -c demo/.tbls.yml -t json -o demo/schema.json
```

## Build the site locally

```bash
go run ./cmd/dbfolio build demo/schema.json -o /tmp/dbfolio-demo
```

## README screenshots

`docs/images/table.png` and `docs/images/relationships.png` are taken from the demo with headless Chromium (`sudo apt-get install chromium fonts-noto-cjk` in the dev container):

```bash
go run ./cmd/dbfolio build demo/schema.json -o /tmp/dbfolio-demo
chromium --headless=new --no-sandbox --hide-scrollbars --window-size=1280,1000 \
  --screenshot=docs/images/table.png "file:///tmp/dbfolio-demo/tables/public.orders.html"
chromium --headless=new --no-sandbox --hide-scrollbars --window-size=1280,800 \
  --screenshot=docs/images/relationships.png "file:///tmp/dbfolio-demo/relationships.html"
```

Note: tbls v1.96.1 lists the values of a PostgreSQL enum type in a different order from their definition (e.g. `order_status`), so the demo shows them in that order.
