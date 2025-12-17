package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

type CreateUserParams struct {
	Email string `json:"email"`
	Role models.UserRole `json:"role"`
	Status models.AccountStatus `json:"status"`
	DisplayName string `json:"display_name"`
	AvatarURL string `json:"avatar_url"`
	MailingAddress models.Address `json:"mailing_address"`
	Preferences json.RawMessage `json:"preferences"`
}

const create_userSQL = `
INSERT INTO users (email, role, status, display_name, avatar_url, mailing_address, preferences)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, role, status, display_name, avatar_url, mailing_address, preferences, email_verified_at, created_at, updated_at;`

func (q *Queries) CreateUser(ctx context.Context, params CreateUserParams) (*models.Users, error) {
	row := q.db.QueryRow(ctx, create_userSQL, params.Email, params.Role, params.Status, params.DisplayName, params.AvatarURL, params.MailingAddress, params.Preferences)

	var result models.Users
	err := row.Scan(&result.ID, &result.Email, &result.Role, &result.Status, &result.DisplayName, &result.AvatarURL, &result.MailingAddress, &result.Preferences, &result.EmailVerifiedAt, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
