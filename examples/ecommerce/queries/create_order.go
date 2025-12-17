package queries

import (
	"context"
	"example/ecommerce/models"
)

type CreateOrderParams struct {
	CustomerID int64 `json:"customer_id"`
	ShippingAddressID int64 `json:"shipping_address_id"`
	BillingAddressID int64 `json:"billing_address_id"`
	SubtotalCents int64 `json:"subtotal_cents"`
	TaxCents int64 `json:"tax_cents"`
	ShippingCents int64 `json:"shipping_cents"`
	TotalCents int64 `json:"total_cents"`
	Notes string `json:"notes"`
}

const create_orderSQL = `
INSERT INTO orders (customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, customer_id, shipping_address_id, billing_address_id, subtotal_cents, tax_cents, shipping_cents, total_cents, notes, created_at, updated_at;`

func (q *Queries) CreateOrder(ctx context.Context, params CreateOrderParams) (*models.Orders, error) {
	row := q.db.QueryRow(ctx, create_orderSQL, params.CustomerID, params.ShippingAddressID, params.BillingAddressID, params.SubtotalCents, params.TaxCents, params.ShippingCents, params.TotalCents, params.Notes)

	var result models.Orders
	err := row.Scan(&result.ID, &result.CustomerID, &result.ShippingAddressID, &result.BillingAddressID, &result.SubtotalCents, &result.TaxCents, &result.ShippingCents, &result.TotalCents, &result.Notes, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
