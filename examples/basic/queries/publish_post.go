package queries

import (
	"context"
)

const publish_postSQL = `
UPDATE posts
SET published = true, published_at = NOW(), updated_at = NOW()
WHERE id = $1;`

func (q *Queries) PublishPost(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, publish_postSQL, id)
	return err
}
