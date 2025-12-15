package queries

import (
	"context"
	"example/basic/models"
)

const get_postsSQL = `
SELECT id, user_id, title, slug, content, published, published_at, created_at, updated_at
FROM posts
WHERE (id = $1 OR $1 IS NULL)
  AND (user_id = $2 OR $2 IS NULL)
  AND (published = $3 OR $3 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetPosts(ctx context.Context, id *int64, user_id *int64, published *bool) ([]models.Posts, error) {
	rows, err := q.db.Query(ctx, get_postsSQL, id, user_id, published)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Posts
	for rows.Next() {
		var item models.Posts
		err := rows.Scan(&item.ID, &item.UserID, &item.Title, &item.Slug, &item.Content, &item.Published, &item.PublishedAt, &item.CreatedAt, &item.UpdatedAt)
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
