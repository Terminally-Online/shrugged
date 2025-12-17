package queries

import (
	"context"
	"example/bookkeeping/models"
	"time"
)

type CreateTicketParams struct {
	UserID int64 `json:"user_id"`
	AssigneeID int64 `json:"assignee_id"`
	Priority models.PriorityLevel `json:"priority"`
	Status models.AccountStatus `json:"status"`
	Title string `json:"title"`
	Description string `json:"description"`
	Tags []string `json:"tags"`
	DueDate time.Time `json:"due_date"`
}

const create_ticketSQL = `
INSERT INTO tickets (user_id, assignee_id, priority, status, title, description, tags, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, assignee_id, priority, status, title, description, tags, due_date, created_at, updated_at, resolved_at;`

func (q *Queries) CreateTicket(ctx context.Context, params CreateTicketParams) (*models.Tickets, error) {
	row := q.db.QueryRow(ctx, create_ticketSQL, params.UserID, params.AssigneeID, params.Priority, params.Status, params.Title, params.Description, params.Tags, params.DueDate)

	var result models.Tickets
	err := row.Scan(&result.ID, &result.UserID, &result.AssigneeID, &result.Priority, &result.Status, &result.Title, &result.Description, &result.Tags, &result.DueDate, &result.CreatedAt, &result.UpdatedAt, &result.ResolvedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
