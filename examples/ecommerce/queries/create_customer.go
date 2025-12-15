package queries

import (
	"context"
	"example/ecommerce/models"
)

const create_customerSQL = `
INSERT INTO customers (email, first_name, last_name, phone)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, last_name, phone, created_at;`

func (q *Queries) CreateCustomer(ctx context.Context, email string, first_name string, last_name string, phone string) (*models.Customers, error) {
	row := q.db.QueryRow(ctx, create_customerSQL, email, first_name, last_name, phone)

	var result models.Customers
	err := row.Scan(&result.ID, &result.Email, &result.FirstName, &result.LastName, &result.Phone, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
