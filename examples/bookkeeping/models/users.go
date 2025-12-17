package models

import (
	"encoding/json"
	"time"
)

type UsersExtension struct{}

type Users struct {
	UsersExtension
	ID              int64           `json:"id"`
	Email           string          `json:"email"`
	Role            UserRole        `json:"role"`
	Status          AccountStatus   `json:"status"`
	DisplayName     string          `json:"display_name"`
	AvatarURL       *string         `json:"avatar_url,omitempty"`
	MailingAddress  *Address        `json:"mailing_address,omitempty"`
	Preferences     json.RawMessage `json:"preferences"`
	EmailVerifiedAt *time.Time      `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       *time.Time      `json:"updated_at,omitempty"`
}
