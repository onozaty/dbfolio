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

-- Column comments ----------------------------------------------------------------

COMMENT ON COLUMN customers.id IS 'Customer ID';
COMMENT ON COLUMN customers.name IS 'Full name';
COMMENT ON COLUMN customers.created_at IS 'When the account was created';

COMMENT ON COLUMN addresses.id IS 'Address ID';
COMMENT ON COLUMN addresses.customer_id IS 'Owner of the address';
COMMENT ON COLUMN addresses.line1 IS 'Street address';
COMMENT ON COLUMN addresses.line2 IS 'Apartment, suite, etc.';
COMMENT ON COLUMN addresses.city IS 'City';
COMMENT ON COLUMN addresses.postal_code IS 'Postal or ZIP code';
COMMENT ON COLUMN addresses.country IS 'ISO 3166-1 alpha-2 country code';

COMMENT ON COLUMN staff.id IS 'Staff ID';
COMMENT ON COLUMN staff.email IS 'Login ID for the back office';
COMMENT ON COLUMN staff.name IS 'Full name';

COMMENT ON COLUMN roles.id IS 'Role ID';
COMMENT ON COLUMN roles.name IS 'Role name, e.g. admin or support';

COMMENT ON COLUMN staff_roles.staff_id IS 'Staff member';
COMMENT ON COLUMN staff_roles.role_id IS 'Granted role';

COMMENT ON COLUMN categories.id IS 'Category ID';
COMMENT ON COLUMN categories.name IS 'Display name';

COMMENT ON COLUMN products.id IS 'Product ID';
COMMENT ON COLUMN products.category_id IS 'Category the product is listed in';
COMMENT ON COLUMN products.name IS 'Display name';
COMMENT ON COLUMN products.description IS 'Product description shown in the store';

COMMENT ON COLUMN inventory.product_id IS 'Product';
COMMENT ON COLUMN inventory.quantity IS 'Units in stock';
COMMENT ON COLUMN inventory.updated_at IS 'Last stock update';

COMMENT ON COLUMN orders.id IS 'Order ID';
COMMENT ON COLUMN orders.customer_id IS 'Customer who placed the order';
COMMENT ON COLUMN orders.shipping_address_id IS 'Where the order is shipped';
COMMENT ON COLUMN orders.status IS 'Order status';
COMMENT ON COLUMN orders.ordered_at IS 'When the order was placed';

COMMENT ON COLUMN order_items.order_id IS 'Order';
COMMENT ON COLUMN order_items.line_no IS 'Line number within the order';
COMMENT ON COLUMN order_items.product_id IS 'Ordered product';
COMMENT ON COLUMN order_items.quantity IS 'Units ordered';

COMMENT ON COLUMN shipments.id IS 'Shipment ID';
COMMENT ON COLUMN shipments.order_id IS 'Order being shipped';
COMMENT ON COLUMN shipments.carrier IS 'Delivery company';
COMMENT ON COLUMN shipments.tracking_no IS 'Tracking number from the carrier';
COMMENT ON COLUMN shipments.shipped_at IS 'When the parcel left the warehouse; null until shipped';

COMMENT ON COLUMN shipment_items.shipment_id IS 'Parcel';
COMMENT ON COLUMN shipment_items.order_id IS 'Order of the line';
COMMENT ON COLUMN shipment_items.line_no IS 'Order line in the parcel';
COMMENT ON COLUMN shipment_items.quantity IS 'Units of the line in the parcel';

COMMENT ON COLUMN reviews.id IS 'Review ID';
COMMENT ON COLUMN reviews.product_id IS 'Reviewed product';
COMMENT ON COLUMN reviews.customer_id IS 'Reviewer';
COMMENT ON COLUMN reviews.body IS 'Review text';
COMMENT ON COLUMN reviews.created_at IS 'When the review was posted';

COMMENT ON COLUMN billing.invoices.id IS 'Invoice ID';
COMMENT ON COLUMN billing.invoices.order_id IS 'Invoiced order';
COMMENT ON COLUMN billing.invoices.issued_at IS 'Issue date';
COMMENT ON COLUMN billing.invoices.amount IS 'Amount to pay';

COMMENT ON COLUMN billing.payments.id IS 'Payment ID';
COMMENT ON COLUMN billing.payments.invoice_id IS 'Invoice paid';
COMMENT ON COLUMN billing.payments.method IS 'How it was paid';
COMMENT ON COLUMN billing.payments.amount IS 'Amount received';
COMMENT ON COLUMN billing.payments.paid_at IS 'When the payment was received';
