package queries

import (
	"context"
	"example/bookkeeping/models"
)

const update_user_roleSQL = `
UPDATE users
SET role = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateUserRole(ctx context.Context, role models.UserRole, id int64) error {
	_, err := q.db.Exec(ctx, update_user_roleSQL, role, id)
	return err
}
