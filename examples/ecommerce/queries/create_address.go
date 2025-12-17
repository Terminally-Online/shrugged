package queries

import (
	"context"
	"example/ecommerce/models"
)

type CreateAddressParams struct {
	CustomerID int64 `json:"customer_id"`
	StreetLine1 string `json:"street_line_1"`
	StreetLine2 string `json:"street_line_2"`
	City string `json:"city"`
	State string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country string `json:"country"`
	IsDefault bool `json:"is_default"`
}

const create_addressSQL = `
INSERT INTO addresses (customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, customer_id, street_line_1, street_line_2, city, state, postal_code, country, is_default, created_at;`

func (q *Queries) CreateAddress(ctx context.Context, params CreateAddressParams) (*models.Addresses, error) {
	row := q.db.QueryRow(ctx, create_addressSQL, params.CustomerID, params.StreetLine1, params.StreetLine2, params.City, params.State, params.PostalCode, params.Country, params.IsDefault)

	var result models.Addresses
	err := row.Scan(&result.ID, &result.CustomerID, &result.StreetLine1, &result.StreetLine2, &result.City, &result.State, &result.PostalCode, &result.Country, &result.IsDefault, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
