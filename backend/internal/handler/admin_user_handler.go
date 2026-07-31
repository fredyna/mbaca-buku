package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type AdminUserHandler struct {
	svc *service.AdminUserService
}

func NewAdminUserHandler(svc *service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{svc: svc}
}

func mapAdminUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCannotModifySelf):
		utils.ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrLastAdmin):
		utils.ErrorResponse(c, http.StatusConflict, "LAST_ADMIN", err.Error())
	case errors.Is(err, service.ErrEmailTaken):
		utils.ErrorResponse(c, http.StatusConflict, "EMAIL_TAKEN", err.Error())
	default:
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
	}
}

func (h *AdminUserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	users, total, err := h.svc.List(c.Request.Context(), page, perPage)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "SERVER_ERROR", err.Error())
		return
	}
	utils.PaginatedResponse(c, users, page, perPage, total)
}

func (h *AdminUserHandler) Create(c *gin.Context) {
	var req dto.AdminUserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		mapAdminUserError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusCreated, user)
}

func (h *AdminUserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.AdminUserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	actorID := c.GetString("user_id")
	user, err := h.svc.Update(c.Request.Context(), id, req, actorID)
	if err != nil {
		mapAdminUserError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, user)
}

func (h *AdminUserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	var req dto.AdminPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), id, req); err != nil {
		mapAdminUserError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "password updated"})
}

func (h *AdminUserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	actorID := c.GetString("user_id")
	if err := h.svc.Delete(c.Request.Context(), id, actorID); err != nil {
		mapAdminUserError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, gin.H{"message": "user deleted"})
}
