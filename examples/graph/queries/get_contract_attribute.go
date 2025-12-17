package queries

import (
	"context"
	"example/graph/models"
)

type GetContractAttributeParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
	ScopeAddress string `json:"scope_address"`
	Name string `json:"name"`
}

const get_contract_attributeSQL = `
SELECT chain_id, contract_address, token_id, scope_address, name, value, block_number
FROM contract_attribute
WHERE chain_id = $1
  AND contract_address = $2
  AND token_id = $3
  AND scope_address = $4
  AND name = $5
ORDER BY block_number DESC
LIMIT 1;`

func (q *Queries) GetContractAttribute(ctx context.Context, params GetContractAttributeParams) (*models.ContractAttribute, error) {
	row := q.db.QueryRow(ctx, get_contract_attributeSQL, params.ChainID, params.ContractAddress, params.TokenID, params.ScopeAddress, params.Name)

	var result models.ContractAttribute
	err := row.Scan(&result.ChainID, &result.ContractAddress, &result.TokenID, &result.ScopeAddress, &result.Name, &result.Value, &result.BlockNumber)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
