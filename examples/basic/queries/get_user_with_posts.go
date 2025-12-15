package queries

import (
	"context"
	"encoding/json"
	"example/basic/models"
)

type GetUserWithPostsRow struct {
	ID    *int64         `json:"id,omitempty"`
	Email *string        `json:"email,omitempty"`
	Name  *string        `json:"name,omitempty"`
	Posts []models.Posts `json:"posts,omitempty"`
}

const get_user_with_postsSQL = `
SELECT
    u.id,
    u.email,
    u.name,
    (SELECT json_agg(p.*) FROM posts p WHERE p.user_id = u.id) as posts
FROM users u
WHERE u.id = $1;`

func (q *Queries) GetUserWithPosts(ctx context.Context, id int64) (*GetUserWithPostsRow, error) {
	row := q.db.QueryRow(ctx, get_user_with_postsSQL, id)

	var result GetUserWithPostsRow
	var postsJSON []byte

	err := row.Scan(&result.ID, &result.Email, &result.Name, &postsJSON)
	if err != nil {
		return nil, err
	}

	if postsJSON != nil {
		if err := json.Unmarshal(postsJSON, &result.Posts); err != nil {
			return nil, err
		}
	}

	return &result, nil
}
