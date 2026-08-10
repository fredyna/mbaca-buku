package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Provider     string    `json:"provider"`
	ProviderID   string    `json:"provider_id"`
	AvatarURL    string    `json:"avatar_url"`
	IsOAuth      bool      `json:"is_oauth"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
