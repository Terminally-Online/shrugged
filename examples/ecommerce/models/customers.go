package models

import (
	"time"
)

type CustomersExtension struct{}

type Customers struct {
	CustomersExtension
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     *string   `json:"phone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
