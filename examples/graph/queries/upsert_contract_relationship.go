package queries

import (
	"context"
)

const upsert_contract_relationshipSQL = `
INSERT INTO contract_relationship (chain_id, contract_address, asset_contract_address, relationship_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (chain_id, contract_address, asset_contract_address, relationship_type) DO NOTHING;`

func (q *Queries) UpsertContractRelationship(ctx context.Context, chain_id int64, contract_address string, asset_contract_address string, relationship_type string) error {
	_, err := q.db.Exec(ctx, upsert_contract_relationshipSQL, chain_id, contract_address, asset_contract_address, relationship_type)
	return err
}
