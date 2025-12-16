package queries

import (
	"context"
	"example/graph/models"
)

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

func (q *Queries) GetContractAttribute(ctx context.Context, chain_id int64, contract_address string, token_id string, scope_address string, name string) (*models.ContractAttribute, error) {
	row := q.db.QueryRow(ctx, get_contract_attributeSQL, chain_id, contract_address, token_id, scope_address, name)

	var result models.ContractAttribute
	err := row.Scan(&result.ChainID, &result.ContractAddress, &result.TokenID, &result.ScopeAddress, &result.Name, &result.Value, &result.BlockNumber)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
