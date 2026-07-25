package utils

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func SuccessResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(code, APIResponse{Success: true, Data: data})
}

func ErrorResponse(c *gin.Context, code int, errCode string, message string) {
	c.JSON(code, APIResponse{Success: false, Error: &APIError{Code: errCode, Message: message}})
}

func PaginatedResponse(c *gin.Context, data interface{}, page, perPage, total int) {
	c.JSON(200, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Page: page, PerPage: perPage, Total: total},
	})
}
