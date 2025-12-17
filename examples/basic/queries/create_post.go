package queries

import (
	"context"
	"example/basic/models"
)

type CreatePostParams struct {
	UserID int64 `json:"user_id"`
	Title string `json:"title"`
	Slug string `json:"slug"`
	Content string `json:"content"`
}

const create_postSQL = `
INSERT INTO posts (user_id, title, slug, content)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, slug, content, published, published_at, created_at, updated_at;`

func (q *Queries) CreatePost(ctx context.Context, params CreatePostParams) (*models.Posts, error) {
	row := q.db.QueryRow(ctx, create_postSQL, params.UserID, params.Title, params.Slug, params.Content)

	var result models.Posts
	err := row.Scan(&result.ID, &result.UserID, &result.Title, &result.Slug, &result.Content, &result.Published, &result.PublishedAt, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
