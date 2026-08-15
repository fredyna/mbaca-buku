package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/supabase"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// activityRecorder records a sign-in for the account that just authenticated.
// Satisfied by *service.ActivityService.
//
// Sign-ins are recorded here rather than in AuthService so that service stays
// free of HTTP concerns: the User-Agent and client address only exist at this
// layer, and AuthService's tests keep working untouched.
type activityRecorder interface {
	RecordAsync(userID, event string, meta model.RequestMeta)
}

type AuthHandler struct {
	authService *service.AuthService
	recorder    activityRecorder
}

func NewAuthHandler(authService *service.AuthService, recorder activityRecorder) *AuthHandler {
	return &AuthHandler{authService: authService, recorder: recorder}
}

// recordLogin logs a successful sign-in. A nil recorder disables it, which lets
// a caller construct the handler without a database and Redis behind it.
func (h *AuthHandler) recordLogin(c *gin.Context, userID string) {
	if h.recorder == nil {
		return
	}
	h.recorder.RecordAsync(userID, model.EventLogin, utils.RequestMetaOf(c))
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

	h.recordLogin(c, resp.User.ID)
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

	h.recordLogin(c, resp.User.ID)
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

// OAuth trades a Supabase access token for this API's own JWT, creating or
// refreshing the matching row in the users table along the way. The response is
// shaped exactly like Login's, so the frontend stores the token the same way
// whichever button the user pressed.
func (h *AuthHandler) OAuth(c *gin.Context) {
	var req dto.OAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.authService.OAuthLogin(c.Request.Context(), req)
	switch {
	case err == nil:
		h.recordLogin(c, resp.User.ID)
		utils.SuccessResponse(c, http.StatusOK, resp)
	case errors.Is(err, supabase.ErrInvalidToken):
		utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, supabase.ErrNotConfigured):
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "OAUTH_NOT_CONFIGURED",
			"Google sign-in is unavailable: set SUPABASE_URL and SUPABASE_ANON_KEY on the API")
	default:
		utils.ErrorResponse(c, http.StatusInternalServerError, "OAUTH_ERROR", err.Error())
	}
}
