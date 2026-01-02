package models

import (
	"encoding/json"
	"time"
)

type AuditLogExtension struct{}

type AuditLog struct {
	ID           int64           `json:"id"`
	UserID       *int64          `json:"user_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *int64          `json:"resource_id,omitempty"`
	OldValues    json.RawMessage `json:"old_values,omitempty"`
	NewValues    json.RawMessage `json:"new_values,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	AuditLogExtension
}
