package queries

import (
	"context"
	"encoding/json"
	"example/ecommerce/models"
)

type CreateProductParams struct {
	CategoryID int64 `json:"category_id"`
	Sku string `json:"sku"`
	Name string `json:"name"`
	Description string `json:"description"`
	PriceCents int64 `json:"price_cents"`
	QuantityInStock int32 `json:"quantity_in_stock"`
	WeightGrams int32 `json:"weight_grams"`
	Metadata json.RawMessage `json:"metadata"`
	Tags []string `json:"tags"`
}

const create_productSQL = `
INSERT INTO products (category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, metadata, tags)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, category_id, sku, name, description, price_cents, quantity_in_stock, weight_grams, is_active, metadata, tags, created_at, updated_at;`

func (q *Queries) CreateProduct(ctx context.Context, params CreateProductParams) (*models.Products, error) {
	row := q.db.QueryRow(ctx, create_productSQL, params.CategoryID, params.Sku, params.Name, params.Description, params.PriceCents, params.QuantityInStock, params.WeightGrams, params.Metadata, params.Tags)

	var result models.Products
	err := row.Scan(&result.ID, &result.CategoryID, &result.Sku, &result.Name, &result.Description, &result.PriceCents, &result.QuantityInStock, &result.WeightGrams, &result.IsActive, &result.Metadata, &result.Tags, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
