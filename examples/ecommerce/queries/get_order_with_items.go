package queries

import (
	"context"
	"encoding/json"
	"example/ecommerce/models"
	"time"
)

type GetOrderWithItemsRow struct {
	ID                *int64              `json:"id,omitempty"`
	CustomerID        *int64              `json:"customer_id,omitempty"`
	ShippingAddressID *int64              `json:"shipping_address_id,omitempty"`
	BillingAddressID  *int64              `json:"billing_address_id,omitempty"`
	SubtotalCents     *int64              `json:"subtotal_cents,omitempty"`
	TaxCents          *int64              `json:"tax_cents,omitempty"`
	ShippingCents     *int64              `json:"shipping_cents,omitempty"`
	TotalCents        *int64              `json:"total_cents,omitempty"`
	Notes             *string             `json:"notes,omitempty"`
	CreatedAt         *time.Time          `json:"created_at,omitempty"`
	UpdatedAt         *time.Time          `json:"updated_at,omitempty"`
	Items             []models.OrderItems `json:"items,omitempty"`
}

const getOrderWithItemsSQL = `
SELECT
    o.id,
    o.customer_id,
    o.shipping_address_id,
    o.billing_address_id,
    o.subtotal_cents,
    o.tax_cents,
    o.shipping_cents,
    o.total_cents,
    o.notes,
    o.created_at,
    o.updated_at,
    (SELECT json_agg(oi.*) FROM order_items oi WHERE oi.order_id = o.id) as items
FROM orders o
WHERE o.id = $1;`

func (q *Queries) GetOrderWithItems(ctx context.Context, id int64) (*GetOrderWithItemsRow, error) {
	row := q.db.QueryRow(ctx, getOrderWithItemsSQL, id)

	var result GetOrderWithItemsRow
	var itemsJSON []byte

	err := row.Scan(&result.ID, &result.CustomerID, &result.ShippingAddressID, &result.BillingAddressID, &result.SubtotalCents, &result.TaxCents, &result.ShippingCents, &result.TotalCents, &result.Notes, &result.CreatedAt, &result.UpdatedAt, &itemsJSON)
	if err != nil {
		return nil, err
	}

	if itemsJSON != nil {
		if err := json.Unmarshal(itemsJSON, &result.Items); err != nil {
			return nil, err
		}
	}

	return &result, nil
}
