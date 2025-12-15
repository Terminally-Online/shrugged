package queries

import (
	"context"
	"example/basic/models"
)

const create_postSQL = `
INSERT INTO posts (user_id, title, slug, content)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, slug, content, published, published_at, created_at, updated_at;`

func (q *Queries) CreatePost(ctx context.Context, user_id int64, title string, slug string, content string) (*models.Posts, error) {
	row := q.db.QueryRow(ctx, create_postSQL, user_id, title, slug, content)

	var result models.Posts
	err := row.Scan(&result.ID, &result.UserID, &result.Title, &result.Slug, &result.Content, &result.Published, &result.PublishedAt, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
