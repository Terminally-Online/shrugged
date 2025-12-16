package queries

import (
	"context"
	"example/graph/models"
)

const get_contract_relationshipsSQL = `
SELECT chain_id, contract_address, asset_contract_address, relationship_type
FROM contract_relationship
WHERE chain_id = $1 AND contract_address = $2;`

func (q *Queries) GetContractRelationships(ctx context.Context, chain_id int64, contract_address string) ([]models.ContractRelationship, error) {
	rows, err := q.db.Query(ctx, get_contract_relationshipsSQL, chain_id, contract_address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ContractRelationship
	for rows.Next() {
		var item models.ContractRelationship
		err := rows.Scan(&item.ChainID, &item.ContractAddress, &item.AssetContractAddress, &item.RelationshipType)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
