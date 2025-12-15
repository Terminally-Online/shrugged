package queries

import (
	"context"
	"example/basic/models"
)

const create_userSQL = `
INSERT INTO users (email, name, bio)
VALUES ($1, $2, $3)
RETURNING id, email, name, bio, created_at, updated_at;`

func (q *Queries) CreateUser(ctx context.Context, email string, name string, bio string) (*models.Users, error) {
	row := q.db.QueryRow(ctx, create_userSQL, email, name, bio)

	var result models.Users
	err := row.Scan(&result.ID, &result.Email, &result.Name, &result.Bio, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
