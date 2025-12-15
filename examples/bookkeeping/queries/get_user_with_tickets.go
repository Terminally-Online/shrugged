package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

type GetUserWithTicketsRow struct {
	ID          *int64           `json:"id,omitempty"`
	Email       *string          `json:"email,omitempty"`
	DisplayName *string          `json:"display_name,omitempty"`
	Role        *models.UserRole `json:"role,omitempty"`
	Tickets     []models.Tickets `json:"tickets,omitempty"`
}

const get_user_with_ticketsSQL = `
SELECT
    u.id,
    u.email,
    u.display_name,
    u.role,
    (SELECT json_agg(t.*) FROM tickets t WHERE t.user_id = u.id) as tickets
FROM users u
WHERE u.id = $1;`

func (q *Queries) GetUserWithTickets(ctx context.Context, id int64) (*GetUserWithTicketsRow, error) {
	row := q.db.QueryRow(ctx, get_user_with_ticketsSQL, id)

	var result GetUserWithTicketsRow
	var ticketsJSON []byte

	err := row.Scan(&result.ID, &result.Email, &result.DisplayName, &result.Role, &ticketsJSON)
	if err != nil {
		return nil, err
	}

	if ticketsJSON != nil {
		if err := json.Unmarshal(ticketsJSON, &result.Tickets); err != nil {
			return nil, err
		}
	}

	return &result, nil
}
