package models

type ContractAttributeExtension struct{}

type ContractAttribute struct {
	ChainID         int64  `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID         string `json:"token_id"`
	ScopeAddress    string `json:"scope_address"`
	Name            string `json:"name"`
	Value           string `json:"value"`
	BlockNumber     int64  `json:"block_number"`
	ContractAttributeExtension
}
