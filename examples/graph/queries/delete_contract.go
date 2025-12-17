package queries

import (
	"context"
)

type DeleteContractParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
}

const delete_contractSQL = `
DELETE FROM contract
WHERE chain_id = $1 AND contract_address = $2 AND token_id = $3;`

func (q *Queries) DeleteContract(ctx context.Context, params DeleteContractParams) error {
	_, err := q.db.Exec(ctx, delete_contractSQL, params.ChainID, params.ContractAddress, params.TokenID)
	return err
}
