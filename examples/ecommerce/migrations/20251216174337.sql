CREATE SEQUENCE addresses_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE categories_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE customers_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE order_items_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE orders_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE SEQUENCE products_id_seq START 1 INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1;

CREATE TABLE customers (
    id bigint NOT NULL DEFAULT nextval('customers_id_seq'::regclass),
    email character varying NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    phone character varying,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT customers_email_key UNIQUE (email),
    CONSTRAINT customers_pkey PRIMARY KEY (id)
);

CREATE TABLE addresses (
    id bigint NOT NULL DEFAULT nextval('addresses_id_seq'::regclass),
    customer_id bigint NOT NULL,
    street_line_1 text NOT NULL,
    street_line_2 text,
    city text NOT NULL,
    state text NOT NULL,
    postal_code character varying NOT NULL,
    country character varying NOT NULL DEFAULT 'US'::character varying,
    is_default boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT addresses_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers (id) ON DELETE CASCADE,
    CONSTRAINT addresses_pkey PRIMARY KEY (id)
);

CREATE TABLE orders (
    id bigint NOT NULL DEFAULT nextval('orders_id_seq'::regclass),
    customer_id bigint NOT NULL,
    shipping_address_id bigint,
    billing_address_id bigint,
    subtotal_cents bigint NOT NULL,
    tax_cents bigint NOT NULL DEFAULT 0,
    shipping_cents bigint NOT NULL DEFAULT 0,
    total_cents bigint NOT NULL,
    notes text,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT orders_billing_address_id_fkey FOREIGN KEY (billing_address_id) REFERENCES addresses (id),
    CONSTRAINT orders_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customers (id),
    CONSTRAINT orders_pkey PRIMARY KEY (id),
    CONSTRAINT orders_shipping_address_id_fkey FOREIGN KEY (shipping_address_id) REFERENCES addresses (id)
);

CREATE TABLE categories (
    id bigint NOT NULL DEFAULT nextval('categories_id_seq'::regclass),
    parent_id bigint,
    name text NOT NULL,
    slug character varying NOT NULL,
    description text,
    CONSTRAINT categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES categories (id) ON DELETE SET NULL,
    CONSTRAINT categories_pkey PRIMARY KEY (id),
    CONSTRAINT categories_slug_key UNIQUE (slug)
);

CREATE TABLE products (
    id bigint NOT NULL DEFAULT nextval('products_id_seq'::regclass),
    category_id bigint,
    sku character varying NOT NULL,
    name text NOT NULL,
    description text,
    price_cents bigint NOT NULL,
    quantity_in_stock integer NOT NULL DEFAULT 0,
    weight_grams integer,
    is_active boolean NOT NULL DEFAULT true,
    metadata jsonb,
    tags text[],
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone,
    CONSTRAINT products_category_id_fkey FOREIGN KEY (category_id) REFERENCES categories (id) ON DELETE SET NULL,
    CONSTRAINT products_pkey PRIMARY KEY (id),
    CONSTRAINT products_sku_key UNIQUE (sku)
);

CREATE TABLE order_items (
    id bigint NOT NULL DEFAULT nextval('order_items_id_seq'::regclass),
    order_id bigint NOT NULL,
    product_id bigint NOT NULL,
    quantity integer NOT NULL,
    unit_price_cents bigint NOT NULL,
    total_cents bigint NOT NULL,
    CONSTRAINT order_items_order_id_fkey FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE,
    CONSTRAINT order_items_pkey PRIMARY KEY (id),
    CONSTRAINT order_items_product_id_fkey FOREIGN KEY (product_id) REFERENCES products (id)
);

CREATE INDEX idx_addresses_customer_id ON addresses (customer_id);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);

CREATE INDEX idx_order_items_product_id ON order_items (product_id);

CREATE INDEX idx_orders_customer_id ON orders (customer_id);

CREATE INDEX idx_products_category_id ON products (category_id);

CREATE INDEX idx_products_is_active ON products (is_active) WHERE (is_active = true);

CREATE INDEX idx_products_tags ON products USING gin (tags);

GRANT USAGE ON TYPE addresses TO PUBLIC;

GRANT USAGE ON TYPE categories TO PUBLIC;

GRANT USAGE ON TYPE customers TO PUBLIC;

GRANT USAGE ON TYPE order_items TO PUBLIC;

GRANT USAGE ON TYPE orders TO PUBLIC;

GRANT USAGE ON TYPE products TO PUBLIC;

COMMENT ON SCHEMA public IS NULL;

