package queries

import (
	"context"
)

type AssignTicketParams struct {
	AssigneeID int64 `json:"assignee_id"`
	ID int64 `json:"id"`
}

const assign_ticketSQL = `
UPDATE tickets
SET assignee_id = $1,
    updated_at = NOW()
WHERE id = $2;`

func (q *Queries) AssignTicket(ctx context.Context, params AssignTicketParams) error {
	_, err := q.db.Exec(ctx, assign_ticketSQL, params.AssigneeID, params.ID)
	return err
}
