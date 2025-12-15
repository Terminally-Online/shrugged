package queries

import (
	"context"
	"example/ecommerce/models"
)

const create_addressSQL = `
INSERT INTO addresses (customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default, created_at;`

func (q *Queries) CreateAddress(ctx context.Context, customer_id int64, street_line_1 string, street_line_2 string, city string, state string, postal_code string, country string, is_default bool) (*models.Addresses, error) {
	row := q.db.QueryRow(ctx, create_addressSQL, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default)

	var result models.Addresses
	err := row.Scan(&result.ID, &result.CustomerID, &result.StreetLine1, &result.StreetLine2, &result.City, &result.State, &result.PostalCode, &result.Country, &result.IsDefault, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
