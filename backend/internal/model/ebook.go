package model

import "time"

type Ebook struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	CoverURL   string    `json:"cover_url"`
	FileURL    string    `json:"file_url"`
	FileSize   int64     `json:"file_size"`
	TotalPages int       `json:"total_pages"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
