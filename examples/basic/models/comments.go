package models

import (
	"time"
)

type CommentsExtension struct{}

type Comments struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	CommentsExtension
}
