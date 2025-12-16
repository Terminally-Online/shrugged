package queries

import (
	"context"
)

const delete_contract_attributeSQL = `
DELETE FROM contract_attribute
WHERE chain_id = $1
  AND contract_address = $2
  AND token_id = $3
  AND scope_address = $4
  AND name = $5;`

func (q *Queries) DeleteContractAttribute(ctx context.Context, chain_id int64, contract_address string, token_id string, scope_address string, name string) error {
	_, err := q.db.Exec(ctx, delete_contract_attributeSQL, chain_id, contract_address, token_id, scope_address, name)
	return err
}
