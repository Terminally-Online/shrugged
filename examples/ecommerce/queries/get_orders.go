package queries

import (
	"context"
	"example/ecommerce/models"
)

const get_ordersSQL = `
SELECT id, customer_id, shipping_address_id, billing_address_id,
       subtotal_cents, tax_cents, shipping_cents, total_cents, notes,
       created_at, updated_at
FROM orders
WHERE (id = $1 OR $1 IS NULL)
  AND (customer_id = $2 OR $2 IS NULL)
ORDER BY created_at DESC;`

func (q *Queries) GetOrders(ctx context.Context, id *int64, customer_id *int64) ([]models.Orders, error) {
	rows, err := q.db.Query(ctx, get_ordersSQL, id, customer_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Orders
	for rows.Next() {
		var item models.Orders
		err := rows.Scan(&item.ID, &item.CustomerID, &item.ShippingAddressID, &item.BillingAddressID, &item.SubtotalCents, &item.TaxCents, &item.ShippingCents, &item.TotalCents, &item.Notes, &item.CreatedAt, &item.UpdatedAt)
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
