package queries

import (
	"context"
	"example/graph/models"
)

const get_contractSQL = `
SELECT chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color
FROM contract
WHERE chain_id = $1 AND contract_address = $2 AND token_id = $3;`

func (q *Queries) GetContract(ctx context.Context, chain_id int64, contract_address string, token_id string) (*models.Contract, error) {
	row := q.db.QueryRow(ctx, get_contractSQL, chain_id, contract_address, token_id)

	var result models.Contract
	err := row.Scan(&result.ChainID, &result.ContractAddress, &result.TokenID, &result.Standard, &result.Protocol, &result.Name, &result.Symbol, &result.Decimals, &result.Icon, &result.Description, &result.Verified, &result.Color)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
