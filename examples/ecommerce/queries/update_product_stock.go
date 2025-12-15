package queries

import (
	"context"
)

const update_product_stockSQL = `
UPDATE products
SET quantity_in_stock = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateProductStock(ctx context.Context, quantity_in_stock int32, id int64) error {
	_, err := q.db.Exec(ctx, update_product_stockSQL, quantity_in_stock, id)
	return err
}
