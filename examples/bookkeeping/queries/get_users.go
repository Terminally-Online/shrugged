package queries

import (
	"context"
	"example/bookkeeping/models"
)

type GetUsersParams struct {
	ID *int64 `json:"id,omitempty"`
	Role *models.UserRole `json:"role,omitempty"`
	Status *models.AccountStatus `json:"status,omitempty"`
}

const get_usersSQL = `
SELECT id, email, role, status, display_name, avatar_url, mailing_address,
       preferences, email_verified_at, created_at, updated_at
FROM users
WHERE (id = $1 OR $1 IS NULL)
  AND (role = $2 OR $2 IS NULL)
  AND (status = $3 OR $3 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetUsers(ctx context.Context, params GetUsersParams) ([]models.Users, error) {
	rows, err := q.db.Query(ctx, get_usersSQL, params.ID, params.Role, params.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Users
	for rows.Next() {
		var item models.Users
		err := rows.Scan(&item.ID, &item.Email, &item.Role, &item.Status, &item.DisplayName, &item.AvatarURL, &item.MailingAddress, &item.Preferences, &item.EmailVerifiedAt, &item.CreatedAt, &item.UpdatedAt)
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
