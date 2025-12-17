package queries

import (
	"context"
	"example/bookkeeping/models"
	"time"
)

type CreateInvoiceParams struct {
	UserID int64 `json:"user_id"`
	Amount models.MoneyAmount `json:"amount"`
	Status models.AccountStatus `json:"status"`
	DueAt time.Time `json:"due_at"`
}

const create_invoiceSQL = `
INSERT INTO invoices (user_id, amount, status, due_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, amount, status, issued_at, due_at, paid_at;`

func (q *Queries) CreateInvoice(ctx context.Context, params CreateInvoiceParams) (*models.Invoices, error) {
	row := q.db.QueryRow(ctx, create_invoiceSQL, params.UserID, params.Amount, params.Status, params.DueAt)

	var result models.Invoices
	err := row.Scan(&result.ID, &result.UserID, &result.Amount, &result.Status, &result.IssuedAt, &result.DueAt, &result.PaidAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
