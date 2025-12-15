package models

import (
	"time"
)

type Invoices struct {
	ID int64 `json:"id"`
	UserID int64 `json:"user_id"`
	Amount MoneyAmount `json:"amount"`
	Status AccountStatus `json:"status"`
	IssuedAt time.Time `json:"issued_at"`
	DueAt time.Time `json:"due_at"`
	PaidAt *time.Time `json:"paid_at"`
}
