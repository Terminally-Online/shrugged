package queries

import (
	"context"
	"example/bookkeeping/models"
	"time"
)

const create_invoiceSQL = `
INSERT INTO invoices (user_id, amount, status, due_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, amount, status, issued_at, due_at, paid_at;`

func (q *Queries) CreateInvoice(ctx context.Context, user_id int64, amount models.MoneyAmount, status models.AccountStatus, due_at time.Time) (*models.Invoices, error) {
	row := q.db.QueryRow(ctx, create_invoiceSQL, user_id, amount, status, due_at)

	var result models.Invoices
	err := row.Scan(&result.ID, &result.UserID, &result.Amount, &result.Status, &result.IssuedAt, &result.DueAt, &result.PaidAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
