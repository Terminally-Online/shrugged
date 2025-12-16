package queries

import (
	"context"
	"example/graph/models"
)

const get_contracts_with_attributeSQL = `
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color,
    CASE WHEN a.name NOT LIKE 'media:%' THEN json_build_object(a.name, a.value)::json ELSE '{}'::json END as attributes,
    CASE WHEN a.name LIKE 'media:%' THEN json_build_object(SUBSTRING(a.name FROM 7), a.value)::json ELSE '{}'::json END as media
FROM contract c
JOIN contract_attribute a ON a.chain_id = c.chain_id AND a.contract_address = c.contract_address AND a.token_id = c.token_id
WHERE c.chain_id = $1
  AND a.name = $2
  AND ($3 = '' OR a.value = $3);`

func (q *Queries) GetContractsWithAttribute(ctx context.Context, chain_id int64, attribute_name string, attribute_value string) ([]models.Contract, error) {
	rows, err := q.db.Query(ctx, get_contracts_with_attributeSQL, chain_id, attribute_name, attribute_value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Contract
	for rows.Next() {
		var item models.Contract
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
