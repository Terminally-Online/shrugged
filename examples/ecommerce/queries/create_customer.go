package queries

import (
	"context"
	"example/ecommerce/models"
)

type CreateCustomerParams struct {
	Email string `json:"email"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Phone string `json:"phone"`
}

const create_customerSQL = `
INSERT INTO customers (email, first_name, last_name, phone)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, last_name, phone, created_at;`

func (q *Queries) CreateCustomer(ctx context.Context, params CreateCustomerParams) (*models.Customers, error) {
	row := q.db.QueryRow(ctx, create_customerSQL, params.Email, params.FirstName, params.LastName, params.Phone)

	var result models.Customers
	err := row.Scan(&result.ID, &result.Email, &result.FirstName, &result.LastName, &result.Phone, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
