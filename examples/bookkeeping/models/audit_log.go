package models

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           int64           `json:"id"`
	UserID       *int64          `json:"user_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *int64          `json:"resource_id"`
	OldValues    json.RawMessage `json:"old_values"`
	NewValues    json.RawMessage `json:"new_values"`
	IPAddress    *string         `json:"ip_address"`
	UserAgent    *string         `json:"user_agent"`
	CreatedAt    time.Time       `json:"created_at"`
}
