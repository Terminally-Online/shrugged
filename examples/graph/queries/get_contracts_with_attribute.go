package queries

import (
	"context"
	"encoding/json"
)

type GetContractsWithAttributeRow struct {
	ChainID         *int64          `json:"chain_id,omitempty"`
	ContractAddress *string         `json:"contract_address,omitempty"`
	TokenID         *string         `json:"token_id,omitempty"`
	Standard        *string         `json:"standard,omitempty"`
	Protocol        *string         `json:"protocol,omitempty"`
	Name            *string         `json:"name,omitempty"`
	Symbol          *string         `json:"symbol,omitempty"`
	Decimals        *int64          `json:"decimals,omitempty"`
	Icon            *string         `json:"icon,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Verified        *bool           `json:"verified,omitempty"`
	Color           *string         `json:"color,omitempty"`
	Attributes      json.RawMessage `json:"attributes,omitempty"`
	Media           json.RawMessage `json:"media,omitempty"`
}

type GetContractsWithAttributeParams struct {
	ChainID        int64  `json:"chain_id"`
	AttributeName  string `json:"attribute_name"`
	AttributeValue string `json:"attribute_value"`
}

const getContractsWithAttributeSQL = `
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color,
    CASE WHEN a.name NOT LIKE 'media:%' THEN json_build_object(a.name, a.value)::json ELSE '{}'::json END as attributes,
    CASE WHEN a.name LIKE 'media:%' THEN json_build_object(SUBSTRING(a.name FROM 7), a.value)::json ELSE '{}'::json END as media
FROM contract c
JOIN contract_attribute a ON a.chain_id = c.chain_id AND a.contract_address = c.contract_address AND a.token_id = c.token_id
WHERE c.chain_id = $1
  AND a.name = $2
  AND ($3 = '' OR a.value = $3);`

func (q *Queries) GetContractsWithAttribute(ctx context.Context, params *GetContractsWithAttributeParams) ([]GetContractsWithAttributeRow, error) {
	rows, err := q.db.Query(ctx, getContractsWithAttributeSQL, params.ChainID, params.AttributeName, params.AttributeValue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GetContractsWithAttributeRow
	for rows.Next() {
		var item GetContractsWithAttributeRow
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.TokenID, &item.Standard, &item.Protocol, &item.Name, &item.Symbol, &item.Decimals, &item.Icon, &item.Description, &item.Verified, &item.Color, &item.Attributes, &item.Media)
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
