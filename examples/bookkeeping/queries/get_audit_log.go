package queries

import (
	"context"
	"example/bookkeeping/models"
)

type GetAuditLogParams struct {
	UserID *int64 `json:"user_id,omitempty"`
	ResourceType *string `json:"resource_type,omitempty"`
	ResourceID *int64 `json:"resource_id,omitempty"`
}

const get_audit_logSQL = `
SELECT id, user_id, action, resource_type, resource_id, old_values, new_values,
       ip_address, user_agent, created_at
FROM audit_log
WHERE (user_id = $1 OR $1 IS NULL)
  AND (resource_type = $2 OR $2 IS NULL)
  AND (resource_id = $3 OR $3 IS NULL)
ORDER BY created_at DESC
LIMIT 100;`

func (q *Queries) GetAuditLog(ctx context.Context, params GetAuditLogParams) ([]models.AuditLog, error) {
	rows, err := q.db.Query(ctx, get_audit_logSQL, params.UserID, params.ResourceType, params.ResourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.AuditLog
	for rows.Next() {
		var item models.AuditLog
		err := rows.Scan(&item.ID, &item.UserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.OldValues, &item.NewValues, &item.IPAddress, &item.UserAgent, &item.CreatedAt)
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
