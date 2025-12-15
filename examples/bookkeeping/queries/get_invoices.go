package queries

import (
	"context"
	"example/bookkeeping/models"
)

const get_invoicesSQL = `
SELECT id, user_id, amount, status, issued_at, due_at, paid_at
FROM invoices
WHERE (id = $1 OR $1 IS NULL)
  AND (user_id = $2 OR $2 IS NULL)
  AND (status = $3 OR $3 IS NULL)
ORDER BY due_at ASC;`

func (q *Queries) GetInvoices(ctx context.Context, id *int64, user_id *int64, status *models.AccountStatus) ([]models.Invoices, error) {
	rows, err := q.db.Query(ctx, get_invoicesSQL, id, user_id, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Invoices
	for rows.Next() {
		var item models.Invoices
		err := rows.Scan(&item.ID, &item.UserID, &item.Amount, &item.Status, &item.IssuedAt, &item.DueAt, &item.PaidAt)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
