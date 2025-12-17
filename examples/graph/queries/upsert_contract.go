package queries

import (
	"context"
)

type UpsertContractParams struct {
	ChainID int64 `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID string `json:"token_id"`
	Standard string `json:"standard"`
	Protocol string `json:"protocol"`
	Name string `json:"name"`
	Symbol string `json:"symbol"`
	Decimals int64 `json:"decimals"`
	Icon string `json:"icon"`
	Description string `json:"description"`
	Verified bool `json:"verified"`
	Color string `json:"color"`
}

const upsert_contractSQL = `
INSERT INTO contract (chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (chain_id, contract_address, token_id)
DO UPDATE SET
    standard = EXCLUDED.standard,
    protocol = EXCLUDED.protocol,
    name = EXCLUDED.name,
    symbol = EXCLUDED.symbol,
    decimals = EXCLUDED.decimals,
    icon = EXCLUDED.icon,
    description = EXCLUDED.description,
    verified = EXCLUDED.verified,
    color = EXCLUDED.color;`

func (q *Queries) UpsertContract(ctx context.Context, params UpsertContractParams) error {
	_, err := q.db.Exec(ctx, upsert_contractSQL, params.ChainID, params.ContractAddress, params.TokenID, params.Standard, params.Protocol, params.Name, params.Symbol, params.Decimals, params.Icon, params.Description, params.Verified, params.Color)
	return err
}
