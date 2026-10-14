-- テスト用サンプルスキーマ (PostgreSQL)
-- 再生成手順は testdata/basic/README.md を参照

DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;

CREATE TABLE regions (
  id   bigint PRIMARY KEY,
  name varchar(100) NOT NULL
);
COMMENT ON TABLE regions IS '地域マスタ';
COMMENT ON COLUMN regions.id IS '地域ID';
COMMENT ON COLUMN regions.name IS '地域名';

CREATE TABLE users (
  id         bigserial PRIMARY KEY,
  name       varchar(100) NOT NULL,
  email      varchar(255) NOT NULL,
  region_id  bigint REFERENCES regions (id),
  status     varchar(20) NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT users_email_key UNIQUE (email),
  CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended', 'deleted'))
);
CREATE INDEX users_region_id_idx ON users (region_id);
COMMENT ON TABLE users IS 'ユーザー情報';
COMMENT ON COLUMN users.id IS 'ユーザーID';
COMMENT ON COLUMN users.name IS '氏名';
COMMENT ON COLUMN users.email IS 'ログインに使用するメールアドレス';
COMMENT ON COLUMN users.region_id IS '地域ID';
COMMENT ON COLUMN users.status IS '状態';
COMMENT ON COLUMN users.created_at IS '登録日時';

CREATE TABLE roles (
  id   bigint PRIMARY KEY,
  name varchar(50) NOT NULL UNIQUE
);
COMMENT ON TABLE roles IS 'ロール';

CREATE TABLE user_roles (
  user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role_id bigint NOT NULL REFERENCES roles (id),
  PRIMARY KEY (user_id, role_id)
);
COMMENT ON TABLE user_roles IS 'ユーザーとロールの対応';

CREATE TABLE orders (
  id         bigserial PRIMARY KEY,
  user_id    bigint NOT NULL REFERENCES users (id),
  ordered_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_user_id_idx ON orders (user_id);
COMMENT ON TABLE orders IS '注文';
COMMENT ON COLUMN orders.user_id IS '注文したユーザー';

-- トリガー
CREATE FUNCTION reject_future_ordered_at() RETURNS trigger AS $$
BEGIN
  IF NEW.ordered_at > now() THEN
    RAISE EXCEPTION 'ordered_at must not be in the future';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orders_check_ordered_at
  BEFORE INSERT OR UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION reject_future_ordered_at();
COMMENT ON TRIGGER orders_check_ordered_at ON orders IS '未来日時の注文を拒否する';

-- 複合主キー
CREATE TABLE order_items (
  order_id   bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
  line_no    integer NOT NULL,
  product    varchar(200) NOT NULL,
  quantity   integer NOT NULL CHECK (quantity > 0),
  PRIMARY KEY (order_id, line_no)
);
COMMENT ON TABLE order_items IS '注文明細';

-- 複合 FK
CREATE TABLE shipments (
  id       bigserial PRIMARY KEY,
  order_id bigint NOT NULL,
  line_no  integer NOT NULL,
  shipped_at timestamptz,
  -- order_id は単独の FK と複合 FK の両方に含まれる
  CONSTRAINT shipments_order_fkey FOREIGN KEY (order_id) REFERENCES orders (id),
  CONSTRAINT shipments_order_item_fkey FOREIGN KEY (order_id, line_no)
    REFERENCES order_items (order_id, line_no)
);
COMMENT ON TABLE shipments IS '出荷 (注文明細単位)';

CREATE TABLE comments (
  id      bigserial PRIMARY KEY,
  user_id bigint NOT NULL REFERENCES users (id),
  body    text NOT NULL
);
COMMENT ON TABLE comments IS 'コメント';

-- FK を張らないテーブル (.tbls.yml で Virtual Relation を定義する)
CREATE TABLE audit_logs (
  id         bigserial PRIMARY KEY,
  user_id    bigint,
  action     varchar(50) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE VIEW active_users AS
  SELECT id, name, email FROM users WHERE status = 'active';
COMMENT ON VIEW active_users IS '有効なユーザー';
