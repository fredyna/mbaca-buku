package dto

import "time"

type UpdateProgressRequest struct {
	Page int `json:"page" binding:"required,min=1"`
}

// SetStatusRequest carries an explicit completion decision made by the reader.
type SetStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=reading completed"`
}

type ProgressResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	Status   string `json:"status"`
}

type OpenBookResponse struct {
	EbookID  string `json:"ebook_id"`
	LastPage int    `json:"last_page"`
	Status   string `json:"status"`
	FileURL  string `json:"file_url"`
}

type HistoryItem struct {
	EbookID        string    `json:"ebook_id"`
	Title          string    `json:"title"`
	Author         string    `json:"author"`
	CoverURL       string    `json:"cover_url"`
	TotalPages     int       `json:"total_pages"`
	LastPage       int       `json:"last_page"`
	Status         string    `json:"status"`
	IsPrivate      bool      `json:"is_private"`
	UploadedBy     string    `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name"`
	CreatedAt      time.Time `json:"created_at"`
	LastOpened     time.Time `json:"last_opened"`
}

type HistoryResponse struct {
	Reading   []HistoryItem `json:"reading"`
	Completed []HistoryItem `json:"completed"`
}
