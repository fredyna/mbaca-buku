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

// UserListItem is a users row joined with its most recent activity, as the admin
// user list needs it. The timestamps are pointers because a user who has never
// signed in has neither, and rendering that as the zero time would read as
// "January 1st, year 1" in the UI.
type UserListItem struct {
	User
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	LastOS       string     `json:"last_os"`
	LastBrowser  string     `json:"last_browser"`
}
