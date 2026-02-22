package queries

import (
	"context"
)

type UpdateUserParams struct {
	Name string `json:"name"`
	Bio  string `json:"bio"`
	ID   int64  `json:"id"`
}

const updateUserSQL = `
UPDATE users
SET name = COALESCE($1, name),
    bio = COALESCE($2, bio),
    updated_at = NOW()
WHERE id = $3;`

func (q *Queries) UpdateUser(ctx context.Context, params UpdateUserParams) error {
	_, err := q.db.Exec(ctx, updateUserSQL, params.Name, params.Bio, params.ID)
	return err
}
