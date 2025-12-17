package queries

import (
	"context"
	"example/graph/models"
)

type GetContractAttributesParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
	ScopeAddress string `json:"scope_address"`
}

const get_contract_attributesSQL = `
SELECT chain_id, contract_address, token_id, scope_address, name, value, block_number
FROM contract_attribute
WHERE chain_id = $1
  AND contract_address = $2
  AND ($3 = '' OR token_id = $3)
  AND ($4 = '' OR scope_address = $4)
ORDER BY name, block_number DESC;`

func (q *Queries) GetContractAttributes(ctx context.Context, params GetContractAttributesParams) ([]models.ContractAttribute, error) {
	rows, err := q.db.Query(ctx, get_contract_attributesSQL, params.ChainID, params.ContractAddress, params.TokenID, params.ScopeAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ContractAttribute
	for rows.Next() {
		var item models.ContractAttribute
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.TokenID, &item.ScopeAddress, &item.Name, &item.Value, &item.BlockNumber)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
