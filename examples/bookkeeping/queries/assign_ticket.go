package queries

import (
	"context"
)

const assign_ticketSQL = `
UPDATE tickets
SET assignee_id = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) AssignTicket(ctx context.Context, assignee_id int64, id int64) error {
	_, err := q.db.Exec(ctx, assign_ticketSQL, assignee_id, id)
	return err
}
