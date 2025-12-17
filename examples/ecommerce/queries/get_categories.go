package queries

import (
	"context"
	"example/ecommerce/models"
)

type GetCategoriesParams struct {
	ID *int64 `json:"id,omitempty"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

const get_categoriesSQL = `
SELECT id, parent_id, name, slug, description
FROM categories
WHERE (id = $1 OR $1 IS NULL)
  AND (parent_id = $2 OR $2 IS NULL)
ORDER BY name;`

func (q *Queries) GetCategories(ctx context.Context, params GetCategoriesParams) ([]models.Categories, error) {
	rows, err := q.db.Query(ctx, get_categoriesSQL, params.ID, params.ParentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Categories
	for rows.Next() {
		var item models.Categories
		err := rows.Scan(&item.ID, &item.ParentID, &item.Name, &item.Slug, &item.Description)
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
