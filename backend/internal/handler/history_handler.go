package handler

import (
	"net/http"

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

func (h *HistoryHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("user_id")

	history, err := h.historyService.GetHistory(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, history)
}
