package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

type GetUserWithTicketsRow struct {
	ID          *int64           `json:"id"`
	Email       *string          `json:"email"`
	DisplayName *string          `json:"display_name"`
	Role        *models.UserRole `json:"role"`
	Tickets     []models.Tickets `json:"tickets"`
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
