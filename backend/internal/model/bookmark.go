package model

import "time"

type Bookmark struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	EbookID    string    `json:"ebook_id"`
	PageNumber int       `json:"page_number"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}
