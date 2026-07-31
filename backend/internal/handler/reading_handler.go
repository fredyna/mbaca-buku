package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type ReadingHandler struct {
	readingService *service.ReadingService
}

func NewReadingHandler(readingService *service.ReadingService) *ReadingHandler {
	return &ReadingHandler{readingService: readingService}
}

func (h *ReadingHandler) OpenBook(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	lastPage, status, err := h.readingService.OpenBook(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.OpenBookResponse{
		EbookID:  ebookID,
		LastPage: lastPage,
		Status:   status,
	})
}

// SetStatus records the reader's explicit decision to finish a book, or to put
// it back in progress.
func (h *ReadingHandler) SetStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	var req dto.SetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	lastPage, err := h.readingService.SetStatus(c.Request.Context(), userID, ebookID, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.ProgressResponse{
		EbookID:  ebookID,
		LastPage: lastPage,
		Status:   req.Status,
	})
}

func (h *ReadingHandler) GetProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	page, status, err := h.readingService.GetProgress(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.ProgressResponse{
		EbookID:  ebookID,
		LastPage: page,
		Status:   status,
	})
}

func (h *ReadingHandler) UpdateProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	var req dto.UpdateProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.readingService.UpdateProgress(c.Request.Context(), userID, ebookID, req.Page); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "progress updated"})
}
