package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_order_with_itemsSQL = `
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

func (q *Queries) GetOrderWithItems(ctx context.Context, id int64) (*models.Orders, error) {
	row := q.db.QueryRow(ctx, get_order_with_itemsSQL, id)

	var result models.Orders
	err := row.Scan(&result.ID, &result.CustomerID, &result.ShippingAddressID, &result.BillingAddressID, &result.SubtotalCents, &result.TaxCents, &result.ShippingCents, &result.TotalCents, &result.Notes, &result.CreatedAt, &result.UpdatedAt, &result.Items)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
