package models

type CategoriesExtension struct{}

type Categories struct {
	CategoriesExtension
	ID          int64   `json:"id"`
	ParentID    *int64  `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
}
