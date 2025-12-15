package models

import (
	"encoding/json"
	"time"
)

type Users struct {
	ID              int64           `json:"id"`
	Email           string          `json:"email"`
	Role            UserRole        `json:"role"`
	Status          AccountStatus   `json:"status"`
	DisplayName     string          `json:"display_name"`
	AvatarURL       *string         `json:"avatar_url"`
	MailingAddress  *Address        `json:"mailing_address"`
	Preferences     json.RawMessage `json:"preferences"`
	EmailVerifiedAt *time.Time      `json:"email_verified_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       *time.Time      `json:"updated_at"`
}
