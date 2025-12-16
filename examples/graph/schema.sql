CREATE TABLE contract (
    chain_id                        BIGINT      NOT NULL,
    contract_address                TEXT        NOT NULL,
    token_id                        TEXT        NOT NULL DEFAULT '',
    standard                        TEXT        NOT NULL DEFAULT '',
    protocol                        TEXT        NOT NULL DEFAULT '',
    name                            TEXT        NOT NULL DEFAULT '',
    symbol                          TEXT        NOT NULL DEFAULT '',
    decimals                        BIGINT      NOT NULL DEFAULT 0,
    icon                            TEXT        NOT NULL DEFAULT '',
    description                     TEXT        NOT NULL DEFAULT '',
    verified                        BOOLEAN     NOT NULL DEFAULT FALSE,
    color                           TEXT        NOT NULL DEFAULT '',
    PRIMARY KEY (chain_id, contract_address, token_id)
);

CREATE TABLE contract_relationship (
    chain_id BIGINT NOT NULL,
    contract_address TEXT NOT NULL,
    asset_contract_address TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    PRIMARY KEY (chain_id, contract_address, asset_contract_address, relationship_type)
);

CREATE TABLE contract_attribute (
    chain_id BIGINT NOT NULL,
    contract_address TEXT NOT NULL,
    token_id TEXT NOT NULL DEFAULT '',
    scope_address TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    block_number BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chain_id, contract_address, token_id, scope_address, name, block_number)
);

