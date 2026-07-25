package model

import "time"

type ReadingProgress struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	EbookID   string    `json:"ebook_id"`
	LastPage  int       `json:"last_page"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}
