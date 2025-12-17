package queries

import (
	"context"
	"example/bookkeeping/models"
)

type UpdateUserStatusParams struct {
	Status models.AccountStatus `json:"status"`
	ID int64 `json:"id"`
}

const update_user_statusSQL = `
UPDATE users
SET status = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateUserStatus(ctx context.Context, params UpdateUserStatusParams) error {
	_, err := q.db.Exec(ctx, update_user_statusSQL, params.Status, params.ID)
	return err
}
