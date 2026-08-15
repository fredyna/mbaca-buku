package model

import "time"

// Event values stored in user_activity_logs.event.
const (
	// EventLogin is one successful sign-in: password, registration or OAuth.
	EventLogin = "login"
	// EventActive is a throttled sign of life from an authenticated request.
	EventActive = "active"
)

// UserActivity is one row of the activity log.
type UserActivity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Event     string    `json:"event"`
	OS        string    `json:"os"`
	Browser   string    `json:"browser"`
	Device    string    `json:"device"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// RequestMeta carries the only parts of an HTTP request the activity log needs.
// It exists so the service layer never depends on gin, and so the values are
// copied out while the request is still alive: activity rows are written from a
// goroutine that outlives the request, and *gin.Context is recycled by then.
type RequestMeta struct {
	IP        string
	UserAgent string
}
