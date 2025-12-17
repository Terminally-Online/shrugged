package queries

import (
	"context"
)

type UpsertContractAttributeParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
	ScopeAddress string `json:"scope_address"`
	Name string `json:"name"`
	Value string `json:"value"`
	BlockNumber int64 `json:"block_number"`
}

const upsert_contract_attributeSQL = `
INSERT INTO contract_attribute (chain_id, contract_address, token_id, scope_address, name, value, block_number)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (chain_id, contract_address, token_id, scope_address, name, block_number)
DO UPDATE SET value = EXCLUDED.value;`

func (q *Queries) UpsertContractAttribute(ctx context.Context, params UpsertContractAttributeParams) error {
	_, err := q.db.Exec(ctx, upsert_contract_attributeSQL, params.ChainID, params.ContractAddress, params.TokenID, params.ScopeAddress, params.Name, params.Value, params.BlockNumber)
	return err
}
