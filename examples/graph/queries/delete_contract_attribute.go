package queries

import (
	"context"
)

type DeleteContractAttributeParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
	ScopeAddress string `json:"scope_address"`
	Name string `json:"name"`
}

const delete_contract_attributeSQL = `
DELETE FROM contract_attribute
WHERE chain_id = $1
  AND contract_address = $2
  AND token_id = $3
  AND scope_address = $4
  AND name = $5;`

func (q *Queries) DeleteContractAttribute(ctx context.Context, params DeleteContractAttributeParams) error {
	_, err := q.db.Exec(ctx, delete_contract_attributeSQL, params.ChainID, params.ContractAddress, params.TokenID, params.ScopeAddress, params.Name)
	return err
}
