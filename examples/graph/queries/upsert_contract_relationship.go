package queries

import (
	"context"
)

type UpsertContractRelationshipParams struct {
	ChainID              int64  `json:"chain_id"`
	ContractAddress      string `json:"contract_address"`
	AssetContractAddress string `json:"asset_contract_address"`
	RelationshipType     string `json:"relationship_type"`
}

const upsertContractRelationshipSQL = `
INSERT INTO contract_relationship (chain_id, contract_address, asset_contract_address, relationship_type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (chain_id, contract_address, asset_contract_address, relationship_type) DO NOTHING;`

func (q *Queries) UpsertContractRelationship(ctx context.Context, params UpsertContractRelationshipParams) error {
	_, err := q.db.Exec(ctx, upsertContractRelationshipSQL, params.ChainID, params.ContractAddress, params.AssetContractAddress, params.RelationshipType)
	return err
}
