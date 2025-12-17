package queries

import (
	"context"
	"example/basic/models"
)

type CreateUserParams struct {
	Email string `json:"email"`
	Name string `json:"name"`
	Bio string `json:"bio"`
}

const create_userSQL = `
INSERT INTO users (email, name, bio)
VALUES ($1, $2, $3)
RETURNING id, email, name, bio, created_at, updated_at;`

func (q *Queries) CreateUser(ctx context.Context, params CreateUserParams) (*models.Users, error) {
	row := q.db.QueryRow(ctx, create_userSQL, params.Email, params.Name, params.Bio)

	var result models.Users
	err := row.Scan(&result.ID, &result.Email, &result.Name, &result.Bio, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
