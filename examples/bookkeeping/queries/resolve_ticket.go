package queries

import (
	"context"
)

const resolveTicketSQL = `
UPDATE tickets
SET status = 'deleted',
    resolved_at = NOW(),
    updated_at = NOW()
WHERE id = $1;`

func (q *Queries) ResolveTicket(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, resolveTicketSQL, id)
	return err
}
