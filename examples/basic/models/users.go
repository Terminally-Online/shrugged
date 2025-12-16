package models

import (
	"time"
)

type UsersExtension struct {}

type Users struct {
	UsersExtension
	ID int64 `json:"id"`
	Email string `json:"email"`
	Name string `json:"name"`
	Bio *string `json:"bio,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
