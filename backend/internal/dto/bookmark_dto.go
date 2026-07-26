package dto

type CreateBookmarkRequest struct {
	PageNumber int    `json:"page_number" binding:"required,min=1"`
	Note       string `json:"note"`
}
