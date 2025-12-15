package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_categoriesSQL = `
SELECT id, parent_id, name, slug, description
FROM categories
WHERE (id = $1 OR $1 IS NULL)
  AND (parent_id = $2 OR $2 IS NULL)
ORDER BY name;`

func (q *Queries) GetCategories(ctx context.Context, id *int64, parent_id *int64) ([]models.Categories, error) {
	rows, err := q.db.Query(ctx, get_categoriesSQL, id, parent_id)
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
