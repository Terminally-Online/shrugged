package models

type ContractRelationship struct {
	ChainID              int64  `json:"chain_id"`
	ContractAddress      string `json:"contract_address"`
	AssetContractAddress string `json:"asset_contract_address"`
	RelationshipType     string `json:"relationship_type"`
}
