package queries

import (
	"context"
)

const mark_invoice_paidSQL = `
UPDATE invoices
SET status = 'active',
    paid_at = NOW()
WHERE id = $1;`

func (q *Queries) MarkInvoicePaid(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, mark_invoice_paidSQL, id)
	return err
}
