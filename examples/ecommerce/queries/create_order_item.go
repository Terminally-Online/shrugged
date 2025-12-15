package queries

import (
	"context"
	"example/ecommerce/models"
)

const create_order_itemSQL = `
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, total_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_id, product_id, quantity, unit_price_cents, total_cents;`

func (q *Queries) CreateOrderItem(ctx context.Context, order_id int64, product_id int64, quantity int32, unit_price_cents int64, total_cents int64) (*models.OrderItems, error) {
	row := q.db.QueryRow(ctx, create_order_itemSQL, order_id, product_id, quantity, unit_price_cents, total_cents)

	var result models.OrderItems
	err := row.Scan(&result.ID, &result.OrderID, &result.ProductID, &result.Quantity, &result.UnitPriceCents, &result.TotalCents)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
