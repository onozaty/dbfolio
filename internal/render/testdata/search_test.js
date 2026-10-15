// Tests for the search logic in assets/search.js. Run by TestSearchJS (node).
"use strict";
const assert = require("node:assert/strict");
const path = require("node:path");
const lib = require(path.join(__dirname, "..", "assets", "search.js"));
const search = (index, query, limit) => lib.search(lib.prepare(index), query, limit);

const index = [
  { k: "t", n: "public.user_roles", c: "ユーザーとロールの対応", h: "tables/public.user_roles.html" },
  { k: "c", n: "user_id", t: "public.orders", y: "bigint", c: "注文したユーザー", h: "tables/public.orders.html#column-1" },
  { k: "t", n: "public.users", c: "ユーザー情報", h: "tables/public.users.html" },
  { k: "c", n: "id", t: "public.users", y: "bigint", c: "ユーザーID", h: "tables/public.users.html#column-0" },
  { k: "c", n: "email", t: "public.users", y: "varchar(255)", c: "ログインに使用するメールアドレス", h: "tables/public.users.html#column-2" },
  { k: "t", n: "public.active_users", y: "VIEW", c: "有効なユーザー", h: "tables/public.active_users.html" },
];
const names = (entries) => entries.map((e) => (e.k === "c" ? e.t + "." + e.n : e.n));
const tests = [];
const test = (name, fn) => tests.push([name, fn]);

test("ranks table prefix, table substring, column prefix, column substring, comment", () => {
  assert.deepEqual(names(search(index, "user")), [
    "public.users",        // table, "users" (last part) starts with "user"; shorter than user_roles
    "public.user_roles",   // table, prefix
    "public.active_users", // table, substring
    "public.orders.user_id", // column, prefix
    "public.users.id",     // column, substring via table name
    "public.users.email",  // column, substring via table name
  ]);
});

test("is case-insensitive", () => {
  assert.deepEqual(names(search(index, "EMAIL")), ["public.users.email"]);
});

test("requires every term (AND)", () => {
  assert.deepEqual(names(search(index, "users id")), ["public.users.id"]);
});

test("matches comments, ranked after names", () => {
  assert.deepEqual(names(search(index, "メール")), ["public.users.email"]);
  assert.deepEqual(names(search(index, "users ログイン")), ["public.users.email"]);
});

test("matches a column by table.column", () => {
  assert.deepEqual(names(search(index, "orders.user")), ["public.orders.user_id"]);
});

test("returns nothing for blank queries and no matches", () => {
  assert.deepEqual(search(index, "   "), []);
  assert.deepEqual(search(index, "nothing"), []);
});

test("limits the number of results", () => {
  const many = Array.from({ length: 50 }, (_, i) => ({ k: "t", n: "t" + i, h: "x" }));
  assert.equal(search(many, "t").length, 20);
  assert.equal(search(many, "t", 5).length, 5);
});

test("keeps the index order for ties", () => {
  // Same rank (prefix) and same length.
  const same = [
    { k: "t", n: "xb", h: "x" },
    { k: "t", n: "xa", h: "x" },
  ];
  assert.deepEqual(names(search(same, "x")), ["xb", "xa"]);
});

test("checks the prefix with the first term only", () => {
  const cols = [
    { k: "c", n: "user_id", t: "orders", h: "x" },
    { k: "c", n: "id", t: "users", h: "x" },
  ];
  // First term "id": a prefix of users.id only.
  assert.deepEqual(names(search(cols, "id user")), ["users.id", "orders.user_id"]);
  // First term "user": a prefix of orders.user_id only.
  assert.deepEqual(names(search(cols, "user id")), ["orders.user_id", "users.id"]);
});

test("handles dots in names and the last dotted part", () => {
  const dotted = [
    { k: "c", n: "a.b", t: "s.t", h: "x" },
    { k: "t", n: "s.tbl", h: "x" },
  ];
  assert.deepEqual(names(search(dotted, "t.a.b")), ["s.t.a.b"]);
  assert.deepEqual(names(search(dotted, "tb")), ["s.tbl"]); // prefix of the last part
  assert.deepEqual(names(search(dotted, "b")), ["s.tbl", "s.t.a.b"]); // table first, then column ("b" is a prefix of the last part of "a.b")
});

test("treats regular expression characters literally", () => {
  const special = [
    { k: "c", n: "price(yen)", t: "t", h: "x" },
    { k: "c", n: "priceXyen", t: "t", h: "x" },
  ];
  assert.deepEqual(names(search(special, "(yen)")), ["t.price(yen)"]);
  assert.deepEqual(names(search(special, ".*")), []);
});

let failed = 0;
for (const [name, fn] of tests) {
  try {
    fn();
  } catch (e) {
    failed++;
    console.error("FAIL: " + name + "\n" + e.message);
  }
}
if (failed > 0) process.exit(1);
console.log("ok " + tests.length + " tests");
