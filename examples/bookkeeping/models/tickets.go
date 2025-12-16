package models

import (
	"time"
)

type TicketsExtension struct {}

type Tickets struct {
	TicketsExtension
	ID int64 `json:"id"`
	UserID int64 `json:"user_id"`
	AssigneeID *int64 `json:"assignee_id,omitempty"`
	Priority PriorityLevel `json:"priority"`
	Status AccountStatus `json:"status"`
	Title string `json:"title"`
	Description *string `json:"description,omitempty"`
	Tags []string `json:"tags,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}
