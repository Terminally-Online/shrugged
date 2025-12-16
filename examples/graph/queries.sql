-- name: UpsertContract :exec
INSERT INTO contract (chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color)
VALUES (@chain_id, @contract_address, @token_id, @standard, @protocol, @name, @symbol, @decimals, @icon, @description, @verified, @color)
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
    color = EXCLUDED.color;

-- name: GetContract :row
SELECT chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color
FROM contract
WHERE chain_id = @chain_id AND contract_address = @contract_address AND token_id = @token_id;

-- name: GetContracts :rows
SELECT chain_id, contract_address, token_id, standard, protocol, name, symbol, decimals, icon, description, verified, color
FROM contract
WHERE chain_id = @chain_id
  AND (@standard = '' OR standard = @standard)
  AND (@protocol = '' OR protocol = @protocol)
  AND (@verified::BOOLEAN IS NULL OR verified = @verified)
ORDER BY name;

-- name: DeleteContract :exec
DELETE FROM contract
WHERE chain_id = @chain_id AND contract_address = @contract_address AND token_id = @token_id;

-- name: UpsertContractRelationship :exec
INSERT INTO contract_relationship (chain_id, contract_address, asset_contract_address, relationship_type)
VALUES (@chain_id, @contract_address, @asset_contract_address, @relationship_type)
ON CONFLICT (chain_id, contract_address, asset_contract_address, relationship_type) DO NOTHING;

-- name: GetContractRelationships :rows
SELECT chain_id, contract_address, asset_contract_address, relationship_type
FROM contract_relationship
WHERE chain_id = @chain_id AND contract_address = @contract_address;

-- name: GetRelatedContracts :rows
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, r.relationship_type
FROM contract_relationship r
JOIN contract c ON c.chain_id = r.chain_id AND c.contract_address = r.asset_contract_address AND c.token_id = ''
WHERE r.chain_id = @chain_id AND r.contract_address = @contract_address;

-- name: GetPairsContainingToken :rows
SELECT DISTINCT cr.contract_address as pair_address
FROM contract_relationship cr
WHERE cr.chain_id = @chain_id
  AND cr.asset_contract_address = @token_address
  AND cr.relationship_type IN ('token:0', 'token:1');

-- name: GetPairTokens :rows
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, r.relationship_type
FROM contract_relationship r
JOIN contract c ON c.chain_id = r.chain_id AND c.contract_address = r.asset_contract_address AND c.token_id = ''
WHERE r.chain_id = @chain_id
  AND r.contract_address = @pair_address
  AND r.relationship_type IN ('token:0', 'token:1')
ORDER BY r.relationship_type;

-- name: DeleteContractRelationship :exec
DELETE FROM contract_relationship
WHERE chain_id = @chain_id
  AND contract_address = @contract_address
  AND asset_contract_address = @asset_contract_address
  AND relationship_type = @relationship_type;

-- name: UpsertContractAttribute :exec
INSERT INTO contract_attribute (chain_id, contract_address, token_id, scope_address, name, value, block_number)
VALUES (@chain_id, @contract_address, @token_id, @scope_address, @name, @value, @block_number)
ON CONFLICT (chain_id, contract_address, token_id, scope_address, name, block_number)
DO UPDATE SET value = EXCLUDED.value;

-- name: GetContractAttributes :rows
SELECT chain_id, contract_address, token_id, scope_address, name, value, block_number
FROM contract_attribute
WHERE chain_id = @chain_id
  AND contract_address = @contract_address
  AND (@token_id = '' OR token_id = @token_id)
  AND (@scope_address = '' OR scope_address = @scope_address)
ORDER BY name, block_number DESC;

-- name: GetContractAttribute :row
SELECT chain_id, contract_address, token_id, scope_address, name, value, block_number
FROM contract_attribute
WHERE chain_id = @chain_id
  AND contract_address = @contract_address
  AND token_id = @token_id
  AND scope_address = @scope_address
  AND name = @name
ORDER BY block_number DESC
LIMIT 1;

-- name: GetContractsWithAttribute :rows
SELECT c.chain_id, c.contract_address, c.token_id, c.standard, c.protocol, c.name, c.symbol, c.decimals, c.icon, c.description, c.verified, c.color, a.value as attribute_value
FROM contract c
JOIN contract_attribute a ON a.chain_id = c.chain_id AND a.contract_address = c.contract_address AND a.token_id = c.token_id
WHERE c.chain_id = @chain_id
  AND a.name = @attribute_name
  AND (@attribute_value = '' OR a.value = @attribute_value);

-- name: DeleteContractAttribute :exec
DELETE FROM contract_attribute
WHERE chain_id = @chain_id
  AND contract_address = @contract_address
  AND token_id = @token_id
  AND scope_address = @scope_address
  AND name = @name;

-- name: ComputeAndStoreTokenPaths :exec
WITH RECURSIVE
pair_graph AS (
    SELECT
        contract_address as pair_address,
        MAX(CASE WHEN relationship_type = 'token:0' THEN LOWER(asset_contract_address) END) as token0,
        MAX(CASE WHEN relationship_type = 'token:1' THEN LOWER(asset_contract_address) END) as token1
    FROM contract_relationship
    WHERE chain_id = @chain_id
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
    WHERE from_token = ANY(@token_addresses::TEXT[])

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
      AND prev.current_token != @target_address
),
shortest_paths AS (
    SELECT DISTINCT ON (start_token) start_token, path
    FROM paths
    WHERE current_token = @target_address
    ORDER BY start_token, depth
)
INSERT INTO contract_attribute (chain_id, contract_address, name, value)
SELECT @chain_id, start_token, 'path:usdc', path::TEXT
FROM shortest_paths
ON CONFLICT (chain_id, contract_address, token_id, scope_address, name, block_number)
DO UPDATE SET value = EXCLUDED.value;

