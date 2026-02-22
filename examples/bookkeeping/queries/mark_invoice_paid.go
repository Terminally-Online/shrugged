package queries

import (
	"context"
)

const markInvoicePaidSQL = `
UPDATE invoices
SET status = 'active',
    paid_at = NOW()
WHERE id = $1;`

func (q *Queries) MarkInvoicePaid(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, markInvoicePaidSQL, id)
	return err
}
