package queries

import (
	"context"
)

type UpdateProductStockParams struct {
	QuantityInStock int32 `json:"quantity_in_stock"`
	ID int64 `json:"id"`
}

const update_product_stockSQL = `
UPDATE products
SET quantity_in_stock = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateProductStock(ctx context.Context, params UpdateProductStockParams) error {
	_, err := q.db.Exec(ctx, update_product_stockSQL, params.QuantityInStock, params.ID)
	return err
}
