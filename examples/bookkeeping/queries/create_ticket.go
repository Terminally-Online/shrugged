package queries

import (
	"context"
	"example/bookkeeping/models"
	"time"
)

const create_ticketSQL = `
INSERT INTO tickets (user_id, assignee_id, priority, status, title, description, tags, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, assignee_id, priority, status, title, description, tags, due_date, created_at, updated_at, resolved_at;`

func (q *Queries) CreateTicket(ctx context.Context, user_id int64, assignee_id int64, priority models.PriorityLevel, status models.AccountStatus, title string, description string, tags []string, due_date time.Time) (*models.Tickets, error) {
	row := q.db.QueryRow(ctx, create_ticketSQL, user_id, assignee_id, priority, status, title, description, tags, due_date)

	var result models.Tickets
	err := row.Scan(&result.ID, &result.UserID, &result.AssigneeID, &result.Priority, &result.Status, &result.Title, &result.Description, &result.Tags, &result.DueDate, &result.CreatedAt, &result.UpdatedAt, &result.ResolvedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
