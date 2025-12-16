package queries

import (
	"context"
)

const delete_contractSQL = `
DELETE FROM contract
WHERE chain_id = $1 AND contract_address = $2 AND token_id = $3;`

func (q *Queries) DeleteContract(ctx context.Context, chain_id int64, contract_address string, token_id string) error {
	_, err := q.db.Exec(ctx, delete_contractSQL, chain_id, contract_address, token_id)
	return err
}
