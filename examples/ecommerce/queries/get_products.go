package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_productsSQL = `
SELECT id, category_id, sku, name, description, price_cents, quantity_in_stock,
       weight_grams, is_active, metadata, tags, created_at, updated_at
FROM products
WHERE (id = $1 OR $1 IS NULL)
  AND (category_id = $2 OR $2 IS NULL)
  AND (is_active = $3 OR $3 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetProducts(ctx context.Context, id *int64, category_id *int64, is_active *bool) ([]models.Products, error) {
	rows, err := q.db.Query(ctx, get_productsSQL, id, category_id, is_active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Products
	for rows.Next() {
		var item models.Products
		err := rows.Scan(&item.ID, &item.CategoryID, &item.Sku, &item.Name, &item.Description, &item.PriceCents, &item.QuantityInStock, &item.WeightGrams, &item.IsActive, &item.Metadata, &item.Tags, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
