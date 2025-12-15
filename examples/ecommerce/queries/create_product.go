package queries

import (
	"context"
	"encoding/json"
	"example/ecommerce/models"
)

const create_productSQL = `
INSERT INTO products (category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, metadata, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, is_active, metadata, tags, created_at, updated_at;`

func (q *Queries) CreateProduct(ctx context.Context, category_id int64, sku string, name string, description string, price_cents int64, quantity_in_stock int32, weight_grams int32, metadata json.RawMessage, tags []string) (*models.Products, error) {
	row := q.db.QueryRow(ctx, create_productSQL, category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, metadata, tags)

	var result models.Products
	err := row.Scan(&result.ID, &result.CategoryID, &result.Sku, &result.Name, &result.Description, &result.PriceCents, &result.QuantityInStock, &result.WeightGrams, &result.IsActive, &result.Metadata, &result.Tags, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
