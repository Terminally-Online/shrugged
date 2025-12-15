package queries

import (
	"context"
)

const verify_user_emailSQL = `
UPDATE users
SET status = 'active',
    email_verified_at = NOW(),
    updated_at = NOW()
WHERE id = $1;`

func (q *Queries) VerifyUserEmail(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, verify_user_emailSQL, id)
	return err
}
