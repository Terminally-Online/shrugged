package models

type Categories struct {
	ID          int64   `json:"id"`
	ParentID    *int64  `json:"parent_id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}
