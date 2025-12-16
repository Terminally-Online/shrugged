package queries

import (
	"context"
	"example/graph/models"
)

const get_contractsSQL = `
SELECT chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color
FROM contract
WHERE chain_id = $1
  AND ($2 = '' OR standard = $2)
  AND ($3 = '' OR protocol = $3)
  AND ($4::BOOLEAN IS NULL OR verified = $4)
ORDER BY name;`

func (q *Queries) GetContracts(ctx context.Context, chain_id int64, standard string, protocol string, verified bool) ([]models.Contract, error) {
	rows, err := q.db.Query(ctx, get_contractsSQL, chain_id, standard, protocol, verified)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Contract
	for rows.Next() {
		var item models.Contract
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.TokenID, &item.Standard, &item.Protocol, &item.Name, &item.Symbol, &item.Decimals, &item.Icon, &item.Description, &item.Verified, &item.Color)
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
