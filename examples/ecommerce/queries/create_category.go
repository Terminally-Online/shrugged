package queries

import (
	"context"
	"example/ecommerce/models"
)

type CreateCategoryParams struct {
	ParentID int64 `json:"parent_id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Description string `json:"description"`
}

const create_categorySQL = `
INSERT INTO categories (parent_id, name, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING id, parent_id, name, slug, description;`

func (q *Queries) CreateCategory(ctx context.Context, params CreateCategoryParams) (*models.Categories, error) {
	row := q.db.QueryRow(ctx, create_categorySQL, params.ParentID, params.Name, params.Slug, params.Description)

	var result models.Categories
	err := row.Scan(&result.ID, &result.ParentID, &result.Name, &result.Slug, &result.Description)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
