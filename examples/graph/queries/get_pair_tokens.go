package queries

import (
	"context"
)

type GetPairTokensRow struct {
	ChainID          *int64  `json:"chain_id,omitempty"`
	ContractAddress  *string `json:"contract_address,omitempty"`
	TokenID          *string `json:"token_id,omitempty"`
	Standard         *string `json:"standard,omitempty"`
	Protocol         *string `json:"protocol,omitempty"`
	Name             *string `json:"name,omitempty"`
	Symbol           *string `json:"symbol,omitempty"`
	Decimals         *int64  `json:"decimals,omitempty"`
	Icon             *string `json:"icon,omitempty"`
	Description      *string `json:"description,omitempty"`
	Verified         *bool   `json:"verified,omitempty"`
	Color            *string `json:"color,omitempty"`
	RelationshipType *string `json:"relationship_type,omitempty"`
}

const get_pair_tokensSQL = `
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, r.relationship_type
FROM contract_relationship r
JOIN contract c ON c.chain_id = r.chain_id AND c.contract_address = r.asset_contract_address AND c.token_id = ''
WHERE r.chain_id = $1
  AND r.contract_address = $2
  AND r.relationship_type IN ('token:0', 'token:1')
ORDER BY r.relationship_type;`

func (q *Queries) GetPairTokens(ctx context.Context, chain_id int64, pair_address string) ([]GetPairTokensRow, error) {
	rows, err := q.db.Query(ctx, get_pair_tokensSQL, chain_id, pair_address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GetPairTokensRow
	for rows.Next() {
		var item GetPairTokensRow
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
