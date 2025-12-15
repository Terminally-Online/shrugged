package models

import (
	"time"
)

type Posts struct {
	ID int64 `json:"id"`
	UserID int64 `json:"user_id"`
	Title string `json:"title"`
	Slug string `json:"slug"`
	Content *string `json:"content"`
	Published bool `json:"published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
