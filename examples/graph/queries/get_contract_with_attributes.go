package queries

import (
	"context"
	"example/graph/models"
)

type GetContractWithAttributesParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
}

const get_contract_with_attributesSQL = `
SELECT
    c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol,
    c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color,
    COALESCE(
        json_object_agg(a.name, a.value) FILTER (WHERE a.name IS NOT NULL AND a.name NOT LIKE 'media:%'),
        '{}'
    )::json as attributes,
    COALESCE(
        json_object_agg(SUBSTRING(a.name FROM 7), a.value) FILTER (WHERE a.name LIKE 'media:%'),
        '{}'
    )::json as media
FROM contract c
LEFT JOIN (
    SELECT DISTINCT ON (chain_id, contract_address, token_id, name)
        chain_id, contract_address, token_id, name, value
    FROM contract_attribute
    WHERE scope_address = ''
    ORDER BY chain_id, contract_address, token_id, name, block_number DESC
) a ON c.chain_id = a.chain_id
    AND c.contract_address = a.contract_address
    AND c.token_id = a.token_id
WHERE c.chain_id = $1
    AND c.contract_address = $2
    AND c.token_id = $3
GROUP BY c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol,
    c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color;`

func (q *Queries) GetContractWithAttributes(ctx context.Context, params GetContractWithAttributesParams) (*models.Contract, error) {
	row := q.db.QueryRow(ctx, get_contract_with_attributesSQL, params.ChainID, params.ContractAddress, params.TokenID)

	var result models.Contract
	err := row.Scan(&result.ChainID, &result.ContractAddress, &result.TokenID, &result.Standard, &result.Protocol, &result.Name, &result.Symbol, &result.Decimals, &result.Icon, &result.Description, &result.Verified, &result.Color, &result.Attributes, &result.Media)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
