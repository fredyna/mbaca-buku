package dto

import "time"

type UpdateProgressRequest struct {
	Page int `json:"page" binding:"required,min=1"`
}

type ProgressResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	Status   string `json:"status"`
}

type OpenBookResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	FileURL  string `json:"file_url"`
}

type HistoryItem struct {
	EbookID    string    `json:"ebook_id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	TotalPages int       `json:"total_pages"`
	LastPage   int       `json:"last_page"`
	Status     string    `json:"status"`
	LastOpened time.Time `json:"last_opened"`
}

type HistoryResponse struct {
	Reading   []HistoryItem `json:"reading"`
	Completed []HistoryItem `json:"completed"`
}
