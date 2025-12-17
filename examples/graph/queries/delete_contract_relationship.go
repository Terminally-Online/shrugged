package queries

import (
	"context"
)

type DeleteContractRelationshipParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	AssetContractAddress string `json:"asset_contract_address"`
	RelationshipType string `json:"relationship_type"`
}

const delete_contract_relationshipSQL = `
DELETE FROM contract_relationship
WHERE chain_id = $1
  AND contract_address = $2
  AND asset_contract_address = $3
  AND relationship_type = $4;`

func (q *Queries) DeleteContractRelationship(ctx context.Context, params DeleteContractRelationshipParams) error {
	_, err := q.db.Exec(ctx, delete_contract_relationshipSQL, params.ChainID, params.ContractAddress, params.AssetContractAddress, params.RelationshipType)
	return err
}
