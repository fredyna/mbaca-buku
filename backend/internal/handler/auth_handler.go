package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusConflict, "REGISTER_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "AUTH_ERROR", err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, resp)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, dto.UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	})
}
