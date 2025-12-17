package queries

import (
	"context"
)

type ComputeAndStoreTokenPathsParams struct {
	ChainID int64 `json:"chain_id"`
	TokenAddresses []string `json:"token_addresses"`
	TargetAddress string `json:"target_address"`
}

const compute_and_store_token_pathsSQL = `
WITH RECURSIVE
pair_graph AS (
    SELECT
        contract_address as pair_address,
        MAX(CASE WHEN relationship_type = 'token:0' THEN LOWER(asset_contract_address) END) as token0,
        MAX(CASE WHEN relationship_type = 'token:1' THEN LOWER(asset_contract_address) END) as token1
    FROM contract_relationship
    WHERE chain_id = $1
      AND relationship_type IN ('token:0', 'token:1')
    GROUP BY contract_address
    HAVING COUNT(*) = 2
),
edges AS (
    SELECT token0 as from_token, token1 as to_token, pair_address FROM pair_graph
    UNION ALL
    SELECT token1 as from_token, token0 as to_token, pair_address FROM pair_graph
),
paths AS (
    SELECT
        from_token as start_token,
        to_token as current_token,
        ARRAY[pair_address] as path,
        ARRAY[from_token, to_token] as visited,
        1 as depth
    FROM edges
    WHERE from_token = ANY($2::TEXT[])

    UNION ALL

    SELECT
        prev.start_token,
        next.to_token as current_token,
        prev.path || next.pair_address,
        prev.visited || next.to_token,
        prev.depth + 1
    FROM paths prev
    JOIN edges next ON next.from_token = prev.current_token
    WHERE NOT next.to_token = ANY(prev.visited)
      AND prev.depth < 3
      AND prev.current_token != $3
),
shortest_paths AS (
    SELECT DISTINCT ON (start_token) start_token, path
    FROM paths
    WHERE current_token = $3
    ORDER BY start_token, depth
)
INSERT INTO contract_attribute (chain_id, contract_address, name, value)
SELECT $1, start_token, 'path:usdc', path::TEXT
FROM shortest_paths
ON CONFLICT (chain_id, contract_address, token_id, scope_address, name, block_number)
DO UPDATE SET value = EXCLUDED.value;`

func (q *Queries) ComputeAndStoreTokenPaths(ctx context.Context, params ComputeAndStoreTokenPathsParams) error {
	_, err := q.db.Exec(ctx, compute_and_store_token_pathsSQL, params.ChainID, params.TokenAddresses, params.TargetAddress)
	return err
}
