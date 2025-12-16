package queries

import (
	"context"
)

const upsert_contract_attributeSQL = `
INSERT INTO contract_attribute (chain_id, contract_address, token_id, scope_address, name, value, block_number)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (chain_id, contract_address, token_id, scope_address, name, block_number)
DO UPDATE SET value = EXCLUDED.value;`

func (q *Queries) UpsertContractAttribute(ctx context.Context, chain_id int64, contract_address string, token_id string, scope_address string, name string, value string, block_number int64) error {
	_, err := q.db.Exec(ctx, upsert_contract_attributeSQL, chain_id, contract_address, token_id, scope_address, name, value, block_number)
	return err
}
