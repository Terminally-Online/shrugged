package queries

import (
	"context"
	"encoding/json"
	"example/basic/models"
)

type GetPostWithCommentsRow struct {
	ID       *int64            `json:"id"`
	Title    *string           `json:"title"`
	Content  *string           `json:"content"`
	Comments []models.Comments `json:"comments"`
}

const get_post_with_commentsSQL = `
SELECT
    p.id,
    p.title,
    p.content,
    (SELECT json_agg(c.*) FROM comments c WHERE c.post_id = p.id) as comments
FROM posts p
WHERE p.id = $1;`

func (q *Queries) GetPostWithComments(ctx context.Context, id int64) (*GetPostWithCommentsRow, error) {
	row := q.db.QueryRow(ctx, get_post_with_commentsSQL, id)

	var result GetPostWithCommentsRow
	var commentsJSON []byte

	err := row.Scan(&result.ID, &result.Title, &result.Content, &commentsJSON)
	if err != nil {
		return nil, err
	}

	if commentsJSON != nil {
		if err := json.Unmarshal(commentsJSON, &result.Comments); err != nil {
			return nil, err
		}
	}

	return &result, nil
}
