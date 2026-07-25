package model

import "time"

type History struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	EbookID  string    `json:"ebook_id"`
	OpenedAt time.Time `json:"opened_at"`
}
