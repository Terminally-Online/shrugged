package queries

import (
	"context"
)

const update_userSQL = `
UPDATE users
SET name = COALESCE($1, name),
    bio = COALESCE($2, bio),
    updated_at = NOW()
WHERE id = $3;`

func (q *Queries) UpdateUser(ctx context.Context, name string, bio string, id int64) error {
	_, err := q.db.Exec(ctx, update_userSQL, name, bio, id)
	return err
}
