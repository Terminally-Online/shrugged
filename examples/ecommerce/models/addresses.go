package models

import (
	"time"
)

type AddressesExtension struct{}

type Addresses struct {
	AddressesExtension
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customer_id"`
	StreetLine1 string    `json:"street_line_1"`
	StreetLine2 *string   `json:"street_line_2,omitempty"`
	City        string    `json:"city"`
	State       string    `json:"state"`
	PostalCode  string    `json:"postal_code"`
	Country     string    `json:"country"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}
