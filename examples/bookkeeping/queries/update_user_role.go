package queries

import (
	"context"
	"example/bookkeeping/models"
)

type UpdateUserRoleParams struct {
	Role models.UserRole `json:"role"`
	ID int64 `json:"id"`
}

const update_user_roleSQL = `
UPDATE users
SET role = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) UpdateUserRole(ctx context.Context, params UpdateUserRoleParams) error {
	_, err := q.db.Exec(ctx, update_user_roleSQL, params.Role, params.ID)
	return err
}
