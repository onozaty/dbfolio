-- Demo schema for the dbfolio demo site: a small online shop (PostgreSQL).
-- Regenerate demo/schema.json as described in demo/README.md.

DROP SCHEMA IF EXISTS public CASCADE;
DROP SCHEMA IF EXISTS billing CASCADE;
CREATE SCHEMA public;
CREATE SCHEMA billing;
COMMENT ON SCHEMA billing IS 'Invoices and payments';

-- Customers and access control ------------------------------------------------

CREATE TABLE customers (
  id          bigserial PRIMARY KEY,
  email       varchar(255) NOT NULL UNIQUE,
  name        varchar(100) NOT NULL,
  country     char(2) NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz
);
COMMENT ON TABLE customers IS 'People who buy from the shop';
COMMENT ON COLUMN customers.email IS 'Login ID and contact address';
COMMENT ON COLUMN customers.country IS 'ISO 3166-1 alpha-2 country code';
COMMENT ON COLUMN customers.deleted_at IS 'Set when the account is closed; the row is kept for order history';

CREATE TABLE addresses (
  id          bigserial PRIMARY KEY,
  customer_id bigint NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
  label       varchar(50) NOT NULL DEFAULT 'home',
  line1       varchar(200) NOT NULL,
  line2       varchar(200),
  city        varchar(100) NOT NULL,
  postal_code varchar(20) NOT NULL,
  country     char(2) NOT NULL,
  CONSTRAINT addresses_customer_label_key UNIQUE (customer_id, label)
);
COMMENT ON TABLE addresses IS 'Shipping and billing addresses of customers';
COMMENT ON COLUMN addresses.label IS 'Name the customer gives the address, unique per customer';

CREATE TABLE staff (
  id         bigserial PRIMARY KEY,
  email      varchar(255) NOT NULL UNIQUE,
  name       varchar(100) NOT NULL,
  manager_id bigint REFERENCES staff (id)
);
COMMENT ON TABLE staff IS 'Shop employees who use the back office';
COMMENT ON COLUMN staff.manager_id IS 'Reporting line; null for the top';

CREATE TABLE roles (
  id   smallint PRIMARY KEY,
  name varchar(50) NOT NULL UNIQUE
);
COMMENT ON TABLE roles IS 'Back office roles';

CREATE TABLE staff_roles (
  staff_id bigint NOT NULL REFERENCES staff (id) ON DELETE CASCADE,
  role_id  smallint NOT NULL REFERENCES roles (id),
  PRIMARY KEY (staff_id, role_id)
);
COMMENT ON TABLE staff_roles IS 'Roles granted to staff';

-- Catalog ---------------------------------------------------------------------

CREATE TABLE categories (
  id        integer PRIMARY KEY,
  parent_id integer REFERENCES categories (id),
  name      varchar(100) NOT NULL,
  slug      varchar(100) NOT NULL UNIQUE
);
COMMENT ON TABLE categories IS 'Product categories, as a tree';
COMMENT ON COLUMN categories.parent_id IS 'Parent category; null for top-level categories';
COMMENT ON COLUMN categories.slug IS 'URL-friendly name';

CREATE TABLE products (
  id          bigserial PRIMARY KEY,
  category_id integer NOT NULL REFERENCES categories (id),
  sku         varchar(40) NOT NULL UNIQUE,
  name        varchar(200) NOT NULL,
  description text,
  price       numeric(10, 2) NOT NULL CHECK (price >= 0),
  active      boolean NOT NULL DEFAULT true
);
CREATE INDEX products_category_id_idx ON products (category_id);
COMMENT ON TABLE products IS 'Items for sale';
COMMENT ON COLUMN products.sku IS 'Stock keeping unit';
COMMENT ON COLUMN products.price IS 'Current unit price; orders keep the price at the time of purchase';
COMMENT ON COLUMN products.active IS 'False hides the product from the store';

CREATE TABLE inventory (
  product_id bigint PRIMARY KEY REFERENCES products (id),
  quantity   integer NOT NULL CHECK (quantity >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE inventory IS 'Stock on hand per product';

-- Orders ------------------------------------------------------------------------

CREATE TYPE order_status AS ENUM ('pending', 'paid', 'shipped', 'delivered', 'cancelled');

CREATE TABLE orders (
  id                  bigserial PRIMARY KEY,
  customer_id         bigint NOT NULL REFERENCES customers (id),
  shipping_address_id bigint NOT NULL REFERENCES addresses (id),
  status              order_status NOT NULL DEFAULT 'pending',
  ordered_at          timestamptz NOT NULL DEFAULT now(),
  total               numeric(12, 2) NOT NULL DEFAULT 0
);
CREATE INDEX orders_customer_id_ordered_at_idx ON orders (customer_id, ordered_at);
COMMENT ON TABLE orders IS 'Orders placed by customers';
COMMENT ON COLUMN orders.total IS 'Sum of the order lines, kept up to date by a trigger';

CREATE TABLE order_items (
  order_id   bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
  line_no    smallint NOT NULL,
  product_id bigint NOT NULL REFERENCES products (id),
  quantity   integer NOT NULL CHECK (quantity > 0),
  unit_price numeric(10, 2) NOT NULL,
  PRIMARY KEY (order_id, line_no)
);
COMMENT ON TABLE order_items IS 'Lines of an order';
COMMENT ON COLUMN order_items.unit_price IS 'Product price when the order was placed';

CREATE TABLE shipments (
  id          bigserial PRIMARY KEY,
  order_id    bigint NOT NULL REFERENCES orders (id),
  carrier     varchar(50) NOT NULL,
  tracking_no varchar(100),
  shipped_at  timestamptz
);
COMMENT ON TABLE shipments IS 'Parcels sent for an order; an order can ship in several parcels';

CREATE TABLE shipment_items (
  shipment_id bigint NOT NULL REFERENCES shipments (id) ON DELETE CASCADE,
  order_id    bigint NOT NULL,
  line_no     smallint NOT NULL,
  quantity    integer NOT NULL CHECK (quantity > 0),
  PRIMARY KEY (shipment_id, order_id, line_no),
  CONSTRAINT shipment_items_order_item_fkey FOREIGN KEY (order_id, line_no)
    REFERENCES order_items (order_id, line_no)
);
COMMENT ON TABLE shipment_items IS 'Which order lines (and how many) are in a parcel';

CREATE TABLE reviews (
  id          bigserial PRIMARY KEY,
  product_id  bigint NOT NULL REFERENCES products (id),
  customer_id bigint NOT NULL REFERENCES customers (id),
  rating      smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body        text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT reviews_product_customer_key UNIQUE (product_id, customer_id)
);
COMMENT ON TABLE reviews IS 'Product reviews; one per customer and product';
COMMENT ON COLUMN reviews.rating IS 'Stars, from 1 to 5';

-- Keep orders.total in sync with the order lines.
CREATE FUNCTION update_order_total() RETURNS trigger AS $$
BEGIN
  UPDATE orders
     SET total = (SELECT coalesce(sum(quantity * unit_price), 0)
                    FROM order_items
                   WHERE order_id = coalesce(NEW.order_id, OLD.order_id))
   WHERE id = coalesce(NEW.order_id, OLD.order_id);
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER order_items_update_total
  AFTER INSERT OR UPDATE OR DELETE ON order_items
  FOR EACH ROW EXECUTE FUNCTION update_order_total();
COMMENT ON TRIGGER order_items_update_total ON order_items IS 'Recalculates orders.total';

-- Activity log (no foreign keys; see the virtual relations in .tbls.yml) --------

CREATE TABLE activity_logs (
  id         bigserial PRIMARY KEY,
  staff_id   bigint,
  order_id   bigint,
  action     varchar(50) NOT NULL,
  detail     jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX activity_logs_created_at_idx ON activity_logs (created_at);

-- Views -------------------------------------------------------------------------

CREATE VIEW active_customers AS
  SELECT id, email, name, country, created_at
    FROM customers
   WHERE deleted_at IS NULL;
COMMENT ON VIEW active_customers IS 'Customers with an open account';

CREATE MATERIALIZED VIEW product_sales AS
  SELECT p.id AS product_id, p.name, sum(i.quantity) AS units_sold, sum(i.quantity * i.unit_price) AS revenue
    FROM products p
    JOIN order_items i ON i.product_id = p.id
    JOIN orders o ON o.id = i.order_id
   WHERE o.status <> 'cancelled'
   GROUP BY p.id, p.name;
COMMENT ON MATERIALIZED VIEW product_sales IS 'Units sold and revenue per product, refreshed nightly';

-- Billing schema ----------------------------------------------------------------

CREATE TYPE billing.payment_method AS ENUM ('card', 'bank_transfer', 'wallet');

CREATE TABLE billing.invoices (
  id         bigserial PRIMARY KEY,
  order_id   bigint NOT NULL UNIQUE REFERENCES public.orders (id),
  number     varchar(20) NOT NULL UNIQUE,
  issued_at  timestamptz NOT NULL DEFAULT now(),
  amount     numeric(12, 2) NOT NULL
);
COMMENT ON TABLE billing.invoices IS 'One invoice per order';
COMMENT ON COLUMN billing.invoices.number IS 'Invoice number shown to the customer';

CREATE TABLE billing.payments (
  id         bigserial PRIMARY KEY,
  invoice_id bigint NOT NULL REFERENCES billing.invoices (id),
  method     billing.payment_method NOT NULL,
  amount     numeric(12, 2) NOT NULL CHECK (amount > 0),
  paid_at    timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE billing.payments IS 'Payments received; an invoice can be paid in several payments';
