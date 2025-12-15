-- name: GetCustomers :rows
SELECT id, email, first_name, last_name, phone, created_at
FROM customers
WHERE (id = @id OR @id IS NULL)
  AND (email = @email OR @email IS NULL)
ORDER BY created_at DESC;

-- name: CreateCustomer :row
INSERT INTO customers (email, first_name, last_name, phone)
VALUES (@email, @first_name, @last_name, @phone)
RETURNING id, email, first_name, last_name, phone, created_at;

-- name: GetProducts :rows
SELECT id, category_id, sku, name, description, price_cents, quantity_in_stock,
       weight_grams, is_active, metadata, tags, created_at, updated_at
FROM products
WHERE (id = @id OR @id IS NULL)
  AND (category_id = @category_id OR @category_id IS NULL)
  AND (is_active = @is_active OR @is_active IS NULL)
ORDER BY created_at DESC;

-- name: CreateProduct :row
INSERT INTO products (category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, metadata, tags)
VALUES (@category_id, @sku, @name, @description, @price_cents, @quantity_in_stock, @weight_grams, @metadata, @tags)
RETURNING id, category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, is_active, metadata, tags, created_at, updated_at;

-- name: UpdateProductStock :exec
UPDATE products
SET quantity_in_stock = @quantity_in_stock,
    updated_at = NOW()
WHERE id = @id;

-- name: GetCategories :rows
SELECT id, parent_id, name, slug, description
FROM categories
WHERE (id = @id OR @id IS NULL)
  AND (parent_id = @parent_id OR @parent_id IS NULL)
ORDER BY name;

-- name: CreateCategory :row
INSERT INTO categories (parent_id, name, slug, description)
VALUES (@parent_id, @name, @slug, @description)
RETURNING id, parent_id, name, slug, description;

-- name: GetOrders :rows
SELECT id, customer_id, shipping_address_id, billing_address_id,
       subtotal_cents, tax_cents, shipping_cents, total_cents, notes,
       created_at, updated_at
FROM orders
WHERE (id = @id OR @id IS NULL)
  AND (customer_id = @customer_id OR @customer_id IS NULL)
ORDER BY created_at DESC;

-- name: CreateOrder :row
INSERT INTO orders (customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes)
VALUES (@customer_id, @shipping_address_id, @billing_address_id, @subtotal_cents, @tax_cents, @shipping_cents, @total_cents, @notes)
RETURNING id, customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes, created_at, updated_at;

-- name: GetOrderWithItems :row
SELECT
    o.id,
    o.customer_id,
    o.total_cents,
    o.created_at,
    (SELECT json_agg(oi.*) FROM order_items oi WHERE oi.order_id = o.id) as items
FROM orders o
WHERE o.id = @id;

-- name: CreateOrderItem :row
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, total_cents)
VALUES (@order_id, @product_id, @quantity, @unit_price_cents, @total_cents)
RETURNING id, order_id, product_id, quantity, unit_price_cents, total_cents;

-- name: GetCustomerAddresses :rows
SELECT id, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default, created_at
FROM addresses
WHERE customer_id = @customer_id
ORDER BY is_default DESC, created_at DESC;

-- name: CreateAddress :row
INSERT INTO addresses (customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default)
VALUES (@customer_id, @street_line_1, @street_line_2, @city, @state, @postal_code, @country, @is_default)
RETURNING id, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default, created_at;
