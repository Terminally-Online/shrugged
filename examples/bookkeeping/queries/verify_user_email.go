package queries

import (
	"context"
)

const verifyUserEmailSQL = `
UPDATE users
SET status = 'active',
    email_verified_at = NOW(),
    updated_at = NOW()
WHERE id = $1;`

func (q *Queries) VerifyUserEmail(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, verifyUserEmailSQL, id)
	return err
}
