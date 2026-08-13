package dto

import "time"

type AdminUserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Activity fields, populated by the list endpoint only. Null means the user
	// has never signed in, which the UI shows differently from "a long time
	// ago"; Create and Update leave them null because a user cannot have
	// activity before they exist.
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	LastOS       string     `json:"last_os"`
	LastBrowser  string     `json:"last_browser"`
}

type AdminUserCreateRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
	Role     string `json:"role" binding:"required,oneof=user admin"`
}

type AdminUserUpdateRequest struct {
	Name  string `json:"name" binding:"required,min=2,max=255"`
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=user admin"`
}

type AdminPasswordResetRequest struct {
	Password string `json:"password" binding:"required,min=5"`
}
