package queries

import (
	"context"
)

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

func (q *Queries) UpsertContract(ctx context.Context, chain_id int64, contract_address string, token_id string, standard string, protocol string, name string, symbol string, decimals int64, icon string, description string, verified bool, color string) error {
	_, err := q.db.Exec(ctx, upsert_contractSQL, chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color)
	return err
}
