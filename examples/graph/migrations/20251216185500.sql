CREATE TABLE contract_relationship (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    asset_contract_address text NOT NULL,
    relationship_type text NOT NULL,
    CONSTRAINT contract_relationship_pkey PRIMARY KEY (chain_id, contract_address, asset_contract_address, relationship_type)
);

CREATE TABLE contract (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    token_id text NOT NULL DEFAULT ''::text,
    standard text NOT NULL DEFAULT ''::text,
    protocol text NOT NULL DEFAULT ''::text,
    name text NOT NULL DEFAULT ''::text,
    symbol text NOT NULL DEFAULT ''::text,
    decimals bigint NOT NULL DEFAULT 0,
    icon text NOT NULL DEFAULT ''::text,
    description text NOT NULL DEFAULT ''::text,
    verified boolean NOT NULL DEFAULT false,
    color text NOT NULL DEFAULT ''::text,
    CONSTRAINT contract_pkey PRIMARY KEY (chain_id, contract_address, token_id)
);

CREATE TABLE contract_attribute (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    token_id text NOT NULL DEFAULT ''::text,
    scope_address text NOT NULL DEFAULT ''::text,
    name text NOT NULL,
    value text NOT NULL,
    block_number bigint NOT NULL DEFAULT 0,
    CONSTRAINT contract_attribute_pkey PRIMARY KEY (chain_id, contract_address, token_id, scope_address, name, block_number)
);

GRANT USAGE ON TYPE contract TO PUBLIC;

GRANT USAGE ON TYPE contract_attribute TO PUBLIC;

GRANT USAGE ON TYPE contract_relationship TO PUBLIC;

COMMENT ON SCHEMA public IS NULL;

