package models

type ContractExtension struct {
	Attributes       *StringMap `json:"attributes,omitempty"`
	Media            *StringMap `json:"media,omitempty"`
	RelationshipType *string    `json:"relationship_type,omitempty"`
}

type Contract struct {
	ContractExtension
	ChainID         int64  `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	TokenID         string `json:"token_id"`
	Standard        string `json:"standard"`
	Protocol        string `json:"protocol"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	Decimals        int64  `json:"decimals"`
	Icon            string `json:"icon"`
	Description     string `json:"description"`
	Verified        bool   `json:"verified"`
	Color           string `json:"color"`
}
