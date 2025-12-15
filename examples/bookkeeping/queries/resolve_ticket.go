package queries

import (
	"context"
)

const resolve_ticketSQL = `
UPDATE tickets
SET status = 'deleted',
    resolved_at = NOW(),
    updated_at = NOW()
WHERE id = $1;`

func (q *Queries) ResolveTicket(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, resolve_ticketSQL, id)
	return err
}
