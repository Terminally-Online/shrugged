package queries

import (
	"context"
	"example/ecommerce/models"
)

type CreateOrderItemParams struct {
	OrderID int64 `json:"order_id"`
	ProductID int64 `json:"product_id"`
	Quantity int32 `json:"quantity"`
	UnitPriceCents int64 `json:"unit_price_cents"`
	TotalCents int64 `json:"total_cents"`
}

const create_order_itemSQL = `
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, total_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_id, product_id, quantity, unit_price_cents, total_cents;`

func (q *Queries) CreateOrderItem(ctx context.Context, params CreateOrderItemParams) (*models.OrderItems, error) {
	row := q.db.QueryRow(ctx, create_order_itemSQL, params.OrderID, params.ProductID, params.Quantity, params.UnitPriceCents, params.TotalCents)

	var result models.OrderItems
	err := row.Scan(&result.ID, &result.OrderID, &result.ProductID, &result.Quantity, &result.UnitPriceCents, &result.TotalCents)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
