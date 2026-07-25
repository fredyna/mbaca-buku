package dto

import "time"

type EbookUploadRequest struct {
	Title      string `form:"title" binding:"required,min=1,max=255"`
	Author     string `form:"author"`
	TotalPages int    `form:"total_pages" binding:"required,min=1"`
}

type EbookUpdateRequest struct {
	Title  string `json:"title" binding:"required,min=1,max=255"`
	Author string `json:"author"`
}

type EbookResponse struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	FileSize   int64     `json:"file_size"`
	TotalPages int       `json:"total_pages"`
	CreatedAt  time.Time `json:"created_at"`
}

type EbookFileResponse struct {
	URL string `json:"url"`
}
