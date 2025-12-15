package models

import (
	"time"
)

type Tickets struct {
	ID          int64         `json:"id"`
	UserID      int64         `json:"user_id"`
	AssigneeID  *int64        `json:"assignee_id"`
	Priority    PriorityLevel `json:"priority"`
	Status      AccountStatus `json:"status"`
	Title       string        `json:"title"`
	Description *string       `json:"description"`
	Tags        []string      `json:"tags"`
	DueDate     *time.Time    `json:"due_date"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   *time.Time    `json:"updated_at"`
	ResolvedAt  *time.Time    `json:"resolved_at"`
}
