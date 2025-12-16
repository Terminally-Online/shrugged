package queries

import (
	"context"
)

const delete_contract_relationshipSQL = `
DELETE FROM contract_relationship
WHERE chain_id = $1
  AND contract_address = $2
  AND asset_contract_address = $3
  AND relationship_type = $4;`

func (q *Queries) DeleteContractRelationship(ctx context.Context, chain_id int64, contract_address string, asset_contract_address string, relationship_type string) error {
	_, err := q.db.Exec(ctx, delete_contract_relationshipSQL, chain_id, contract_address, asset_contract_address, relationship_type)
	return err
}
