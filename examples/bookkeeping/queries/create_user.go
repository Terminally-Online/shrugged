package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

const create_userSQL = `
INSERT INTO users (email, role, status, display_name, avatar_url, mailing_address, preferences)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, email, role, status, display_name, avatar_url, mailing_address, preferences, email_verified_at, created_at, updated_at;`

func (q *Queries) CreateUser(ctx context.Context, email string, role models.UserRole, status models.AccountStatus, display_name string, avatar_url string, mailing_address models.Address, preferences json.RawMessage) (*models.Users, error) {
	row := q.db.QueryRow(ctx, create_userSQL, email, role, status, display_name, avatar_url, mailing_address, preferences)

	var result models.Users
	err := row.Scan(&result.ID, &result.Email, &result.Role, &result.Status, &result.DisplayName, &result.AvatarURL, &result.MailingAddress, &result.Preferences, &result.EmailVerifiedAt, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
