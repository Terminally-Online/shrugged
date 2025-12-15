package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_customersSQL = `
SELECT id, email, first_name, last_name, phone, created_at
FROM customers
WHERE (id = $1 OR $1 IS NULL)
  AND (email = $2 OR $2 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetCustomers(ctx context.Context, id *int64, email *string) ([]models.Customers, error) {
	rows, err := q.db.Query(ctx, get_customersSQL, id, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Customers
	for rows.Next() {
		var item models.Customers
		err := rows.Scan(&item.ID, &item.Email, &item.FirstName, &item.LastName, &item.Phone, &item.CreatedAt)
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
