package models

import (
	"time"
)

type Users struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Bio       *string    `json:"bio,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
