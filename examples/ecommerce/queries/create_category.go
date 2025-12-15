package queries

import (
	"context"
	"example/ecommerce/models"
)

const create_categorySQL = `
INSERT INTO categories (parent_id, name, slug, description)
VALUES ($1, $2, $3, $4)
RETURNING id, parent_id, name, slug, description;`

func (q *Queries) CreateCategory(ctx context.Context, parent_id int64, name string, slug string, description string) (*models.Categories, error) {
	row := q.db.QueryRow(ctx, create_categorySQL, parent_id, name, slug, description)

	var result models.Categories
	err := row.Scan(&result.ID, &result.ParentID, &result.Name, &result.Slug, &result.Description)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
