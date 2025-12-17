package queries

import (
	"context"
	"example/bookkeeping/models"
)

type GetTicketsParams struct {
	ID *int64 `json:"id,omitempty"`
	UserID *int64 `json:"user_id,omitempty"`
	AssigneeID *int64 `json:"assignee_id,omitempty"`
	Priority *models.PriorityLevel `json:"priority,omitempty"`
	Status *models.AccountStatus `json:"status,omitempty"`
}

const get_ticketsSQL = `
SELECT id, user_id, assignee_id, priority, status, title, description,
       tags, due_date, created_at, updated_at, resolved_at
FROM tickets
WHERE (id = $1 OR $1 IS NULL)
  AND (user_id = $2 OR $2 IS NULL)
  AND (assignee_id = $3 OR $3 IS NULL)
  AND (priority = $4 OR $4 IS NULL)
  AND (status = $5 OR $5 IS NULL)
ORDER BY
    CASE priority
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
    END,
    created_at DESC;`

func (q *Queries) GetTickets(ctx context.Context, params GetTicketsParams) ([]models.Tickets, error) {
	rows, err := q.db.Query(ctx, get_ticketsSQL, params.ID, params.UserID, params.AssigneeID, params.Priority, params.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Tickets
	for rows.Next() {
		var item models.Tickets
		err := rows.Scan(&item.ID, &item.UserID, &item.AssigneeID, &item.Priority, &item.Status, &item.Title, &item.Description, &item.Tags, &item.DueDate, &item.CreatedAt, &item.UpdatedAt, &item.ResolvedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
