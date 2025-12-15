package queries

import (
	"context"
	"example/bookkeeping/models"
)

const update_user_statusSQL = `
UPDATE users
SET status = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateUserStatus(ctx context.Context, status models.AccountStatus, id int64) error {
	_, err := q.db.Exec(ctx, update_user_statusSQL, status, id)
	return err
}
