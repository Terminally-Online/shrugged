package queries

import (
	"context"
	"example/graph/models"
)

type GetRelatedContractsParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
}

const get_related_contractsSQL = `
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, r.relationship_type
FROM contract_relationship r
JOIN contract c ON c.chain_id = r.chain_id AND c.contract_address = r.asset_contract_address AND c.token_id = ''
WHERE r.chain_id = $1 AND r.contract_address = $2;`

func (q *Queries) GetRelatedContracts(ctx context.Context, params GetRelatedContractsParams) ([]models.Contract, error) {
	rows, err := q.db.Query(ctx, get_related_contractsSQL, params.ChainID, params.ContractAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Contract
	for rows.Next() {
		var item models.Contract
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.TokenID, &item.Standard, &item.Protocol, &item.Name, &item.Symbol, &item.Decimals, &item.Icon, &item.Description, &item.Verified, &item.Color, &item.RelationshipType)
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
