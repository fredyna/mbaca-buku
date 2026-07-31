package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type EbookHandler struct {
	ebookService *service.EbookService
}

func NewEbookHandler(ebookService *service.EbookService) *EbookHandler {
	return &EbookHandler{ebookService: ebookService}
}

func mapEbookError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		utils.ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}
}

func (h *EbookHandler) Upload(c *gin.Context) {
	var req dto.EbookUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()

	userID := c.GetString("user_id")
	fileName := uuid.New().String() + ".pdf"

	ebook, err := h.ebookService.Upload(c.Request.Context(), req, file, header.Size, fileName, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPDF) {
			utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "UPLOAD_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, ebook)
}

func (h *EbookHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "admin"

	ebook, err := h.ebookService.GetByID(c.Request.Context(), id, userID, isAdmin)
	if err != nil {
		mapEbookError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, ebook)
}

func (h *EbookHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	userID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "admin"

	ebooks, total, err := h.ebookService.List(c.Request.Context(), page, perPage, userID, isAdmin)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.PaginatedResponse(c, ebooks, page, perPage, total)
}

func (h *EbookHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.EbookUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	userID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "admin"

	ebook, err := h.ebookService.Update(c.Request.Context(), id, req, userID, isAdmin)
	if err != nil {
		mapEbookError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, ebook)
}

func (h *EbookHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "admin"

	if err := h.ebookService.Delete(c.Request.Context(), id, userID, isAdmin); err != nil {
		mapEbookError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "ebook deleted"})
}

func (h *EbookHandler) GetFileURL(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	isAdmin := c.GetString("role") == "admin"

	url, err := h.ebookService.GetFileURL(c.Request.Context(), id, userID, isAdmin)
	if err != nil {
		mapEbookError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, dto.EbookFileResponse{URL: url})
}
