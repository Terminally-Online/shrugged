package models

import (
	"time"
)

type Customers struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     *string   `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
}
