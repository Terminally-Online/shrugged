package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

type CreateAuditLogParams struct {
	UserID int64 `json:"user_id"`
	Action string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID int64 `json:"resource_id"`
	OldValues json.RawMessage `json:"old_values"`
	NewValues json.RawMessage `json:"new_values"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

const create_audit_logSQL = `
INSERT INTO audit_log (user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, created_at;`

func (q *Queries) CreateAuditLog(ctx context.Context, params CreateAuditLogParams) (*models.AuditLog, error) {
	row := q.db.QueryRow(ctx, create_audit_logSQL, params.UserID, params.Action, params.ResourceType, params.ResourceID, params.OldValues, params.NewValues, params.IPAddress, params.UserAgent)

	var result models.AuditLog
	err := row.Scan(&result.ID, &result.UserID, &result.Action, &result.ResourceType, &result.ResourceID, &result.OldValues, &result.NewValues, &result.IPAddress, &result.UserAgent, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
