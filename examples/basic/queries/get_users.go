package queries

import (
	"context"
	"example/basic/models"
)

type GetUsersParams struct {
	ID *int64 `json:"id,omitempty"`
	Email *string `json:"email,omitempty"`
}

const get_usersSQL = `
SELECT id, email, name, bio, created_at, updated_at
FROM users
WHERE (id = $1 OR $1 IS NULL)
  AND (email = $2 OR $2 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetUsers(ctx context.Context, params GetUsersParams) ([]models.Users, error) {
	rows, err := q.db.Query(ctx, get_usersSQL, params.ID, params.Email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Users
	for rows.Next() {
		var item models.Users
		err := rows.Scan(&item.ID, &item.Email, &item.Name, &item.Bio, &item.CreatedAt, &item.UpdatedAt)
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
