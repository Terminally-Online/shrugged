package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_customer_addressesSQL = `
SELECT id, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default, created_at
FROM addresses
WHERE customer_id = $1
ORDER BY is_default DESC, created_at DESC;`

func (q *Queries) GetCustomerAddresses(ctx context.Context, customer_id int64) ([]models.Addresses, error) {
	rows, err := q.db.Query(ctx, get_customer_addressesSQL, customer_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Addresses
	for rows.Next() {
		var item models.Addresses
		err := rows.Scan(&item.ID, &item.CustomerID, &item.StreetLine1, &item.StreetLine2, &item.City, &item.State, &item.PostalCode, &item.Country, &item.IsDefault, &item.CreatedAt)
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
