package queries

import (
	"context"
	"encoding/json"
	"example/ecommerce/models"
	"time"
)

type GetOrderWithItemsRow struct {
	ID         *int64              `json:"id,omitempty"`
	CustomerID *int64              `json:"customer_id,omitempty"`
	TotalCents *int64              `json:"total_cents,omitempty"`
	CreatedAt  *time.Time          `json:"created_at,omitempty"`
	Items      []models.OrderItems `json:"items,omitempty"`
}

const get_order_with_itemsSQL = `
SELECT
    o.id,
    o.customer_id,
    o.total_cents,
    o.created_at,
    (SELECT json_agg(oi.*) FROM order_items oi WHERE oi.order_id = o.id) as items
FROM orders o
WHERE o.id = $1;`

func (q *Queries) GetOrderWithItems(ctx context.Context, id int64) (*GetOrderWithItemsRow, error) {
	row := q.db.QueryRow(ctx, get_order_with_itemsSQL, id)

	var result GetOrderWithItemsRow
	var itemsJSON []byte

	err := row.Scan(&result.ID, &result.CustomerID, &result.TotalCents, &result.CreatedAt, &itemsJSON)
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
