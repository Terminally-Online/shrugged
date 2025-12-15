package queries

import (
	"context"
	"encoding/json"
	"example/bookkeeping/models"
)

const create_audit_logSQL = `
INSERT INTO audit_log (user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent, created_at;`

func (q *Queries) CreateAuditLog(ctx context.Context, user_id int64, action string, resource_type string, resource_id int64, old_values json.RawMessage, new_values json.RawMessage, ip_address string, user_agent string) (*models.AuditLog, error) {
	row := q.db.QueryRow(ctx, create_audit_logSQL, user_id, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent)

	var result models.AuditLog
	err := row.Scan(&result.ID, &result.UserID, &result.Action, &result.ResourceType, &result.ResourceID, &result.OldValues, &result.NewValues, &result.IPAddress, &result.UserAgent, &result.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
