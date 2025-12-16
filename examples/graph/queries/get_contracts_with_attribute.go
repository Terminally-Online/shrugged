package queries

import (
	"context"
)

type GetContractsWithAttributeRow struct {
	ChainID         *int64  `json:"chain_id,omitempty"`
	ContractAddress *string `json:"contract_address,omitempty"`
	TokenID         *string `json:"token_id,omitempty"`
	Standard        *string `json:"standard,omitempty"`
	Protocol        *string `json:"protocol,omitempty"`
	Name            *string `json:"name,omitempty"`
	Symbol          *string `json:"symbol,omitempty"`
	Decimals        *int64  `json:"decimals,omitempty"`
	Icon            *string `json:"icon,omitempty"`
	Description     *string `json:"description,omitempty"`
	Verified        *bool   `json:"verified,omitempty"`
	Color           *string `json:"color,omitempty"`
	AttributeValue  *string `json:"attribute_value,omitempty"`
}

const get_contracts_with_attributeSQL = `
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, a.value as attribute_value
FROM contract c
JOIN contract_attribute a ON a.chain_id = c.chain_id AND a.contract_address = c.contract_address AND a.token_id = c.token_id
WHERE c.chain_id = $1
  AND a.name = $2
  AND ($3 = '' OR a.value = $3);`

func (q *Queries) GetContractsWithAttribute(ctx context.Context, chain_id int64, attribute_name string, attribute_value string) ([]GetContractsWithAttributeRow, error) {
	rows, err := q.db.Query(ctx, get_contracts_with_attributeSQL, chain_id, attribute_name, attribute_value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GetContractsWithAttributeRow
	for rows.Next() {
		var item GetContractsWithAttributeRow
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.TokenID, &item.Standard, &item.Protocol, &item.Name, &item.Symbol, &item.Decimals, &item.Icon, &item.Description, &item.Verified, &item.Color, &item.AttributeValue)
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
