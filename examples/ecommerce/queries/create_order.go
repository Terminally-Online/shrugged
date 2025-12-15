package queries

import (
	"context"
	"example/ecommerce/models"
)

const create_orderSQL = `
INSERT INTO orders (customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes, created_at, updated_at;`

func (q *Queries) CreateOrder(ctx context.Context, customer_id int64, shipping_address_id int64, billing_address_id int64, subtotal_cents int64, tax_cents int64, shipping_cents int64, total_cents int64, notes string) (*models.Orders, error) {
	row := q.db.QueryRow(ctx, create_orderSQL, customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes)

	var result models.Orders
	err := row.Scan(&result.ID, &result.CustomerID, &result.ShippingAddressID, &result.BillingAddressID, &result.SubtotalCents, &result.TaxCents, &result.ShippingCents, &result.TotalCents, &result.Notes, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
