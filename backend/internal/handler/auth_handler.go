package handler

import (
	"errors"
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

// ChangePassword updates the password of the authenticated user. Validation
// failures answer 400 rather than 401: the frontend logs the user out on any
// 401, which would turn a mistyped old password into an unexpected logout.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), c.GetString("user_id"), req)
	switch {
	case err == nil:
		utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "password updated"})
	case errors.Is(err, service.ErrInvalidOldPassword):
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_OLD_PASSWORD", err.Error())
	case errors.Is(err, service.ErrSamePassword):
		utils.ErrorResponse(c, http.StatusBadRequest, "SAME_PASSWORD", err.Error())
	default:
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
	}
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
		Role:  user.Role,
	})
}
