package dto

import "time"

// EbookUploadRequest carries the metadata a client may set on upload.
// TotalPages is deliberately absent: it is read from the PDF itself so the
// dashboard and the reader always compute progress from the same number.
type EbookUploadRequest struct {
	Title  string `form:"title" binding:"required,min=1,max=255"`
	Author string `form:"author"`
	// Optional; when absent the handler defaults it to true.
	IsPrivate *bool `form:"is_private"`
}

type EbookUpdateRequest struct {
	Title     string `json:"title" binding:"required,min=1,max=255"`
	Author    string `json:"author"`
	IsPrivate *bool  `json:"is_private"`
}

// EbookListQuery carries the paging, search, and sort options of a list
// request. Every field is optional; unset or unrecognised values fall back to
// defaults further down so a stale bookmark still renders a page.
type EbookListQuery struct {
	Page       int
	PerPage    int
	Query      string // keyword matched against the title
	Author     string // exact author match
	Visibility string // "all", "public", or "private"
	Sort       string // "title" or "latest"
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
