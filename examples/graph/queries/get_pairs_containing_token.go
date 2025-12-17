package queries

import (
	"context"
)

type GetPairsContainingTokenRow struct {
	PairAddress *string `json:"pair_address,omitempty"`
}

type GetPairsContainingTokenParams struct {
	ChainID int64 `json:"chain_id"`
	TokenAddress string `json:"token_address"`
}

const get_pairs_containing_tokenSQL = `
SELECT DISTINCT cr.contract_address as pair_address
FROM contract_relationship cr
WHERE cr.chain_id = $1
  AND cr.asset_contract_address = $2
  AND cr.relationship_type IN ('token:0', 'token:1');`

func (q *Queries) GetPairsContainingToken(ctx context.Context, params GetPairsContainingTokenParams) ([]GetPairsContainingTokenRow, error) {
	rows, err := q.db.Query(ctx, get_pairs_containing_tokenSQL, params.ChainID, params.TokenAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GetPairsContainingTokenRow
	for rows.Next() {
		var item GetPairsContainingTokenRow
		err := rows.Scan(&item.PairAddress)
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
