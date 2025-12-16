COMMENT ON SCHEMA public IS 'standard public schema';

REVOKE USAGE ON TYPE products FROM PUBLIC;

REVOKE USAGE ON TYPE orders FROM PUBLIC;

REVOKE USAGE ON TYPE order_items FROM PUBLIC;

REVOKE USAGE ON TYPE customers FROM PUBLIC;

REVOKE USAGE ON TYPE categories FROM PUBLIC;

REVOKE USAGE ON TYPE addresses FROM PUBLIC;

DROP INDEX idx_products_tags;

DROP INDEX idx_products_is_active;

DROP INDEX idx_products_category_id;

DROP INDEX idx_orders_customer_id;

DROP INDEX idx_order_items_product_id;

DROP INDEX idx_order_items_order_id;

DROP INDEX idx_addresses_customer_id;

DROP TABLE order_items;

DROP TABLE products;

DROP TABLE categories;

DROP TABLE orders;

DROP TABLE addresses;

DROP TABLE customers;

DROP SEQUENCE products_id_seq;

DROP SEQUENCE orders_id_seq;

DROP SEQUENCE order_items_id_seq;

DROP SEQUENCE customers_id_seq;

DROP SEQUENCE categories_id_seq;

DROP SEQUENCE addresses_id_seq;

