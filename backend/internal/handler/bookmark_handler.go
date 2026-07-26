package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type BookmarkHandler struct {
	bookmarkService *service.BookmarkService
}

func NewBookmarkHandler(bookmarkService *service.BookmarkService) *BookmarkHandler {
	return &BookmarkHandler{bookmarkService: bookmarkService}
}

func (h *BookmarkHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	var req dto.CreateBookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	bookmark, err := h.bookmarkService.Create(c.Request.Context(), userID, ebookID, req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusConflict, "BOOKMARK_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, bookmark)
}

func (h *BookmarkHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	ebookID := c.Param("id")

	bookmarks, err := h.bookmarkService.List(c.Request.Context(), userID, ebookID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, bookmarks)
}

func (h *BookmarkHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	if err := h.bookmarkService.Delete(c.Request.Context(), id, userID); err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "bookmark deleted"})
}
