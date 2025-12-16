package queries

import (
	"context"
)

type GetPairsContainingTokenRow struct {
	PairAddress *string `json:"pair_address,omitempty"`
}

const get_pairs_containing_tokenSQL = `
SELECT DISTINCT cr.contract_address as pair_address
FROM contract_relationship cr
WHERE cr.chain_id = $1
  AND cr.asset_contract_address = $2
  AND cr.relationship_type IN ('token:0', 'token:1');`

func (q *Queries) GetPairsContainingToken(ctx context.Context, chain_id int64, token_address string) ([]GetPairsContainingTokenRow, error) {
	rows, err := q.db.Query(ctx, get_pairs_containing_tokenSQL, chain_id, token_address)
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
