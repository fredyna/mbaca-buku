package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type HistoryHandler struct {
	historyService *service.HistoryService
}

func NewHistoryHandler(historyService *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{historyService: historyService}
}

func queryIntParam(c *gin.Context, key string) int {
	v, _ := strconv.Atoi(c.Query(key))
	return v
}

func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	status := c.Query("status")

	if status != "" {
		if status != "reading" && status != "completed" {
			utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", "status must be reading or completed")
			return
		}

		page := queryIntParam(c, "page")
		perPage := queryIntParam(c, "per_page")

		history, total, err := h.historyService.List(c.Request.Context(), userID, status, page, perPage)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
			return
		}

		page, perPage = service.NormalizePaging(page, perPage)
		utils.PaginatedResponse(c, history, page, perPage, total)
		return
	}

	history, err := h.historyService.GetHistory(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, history)
}
