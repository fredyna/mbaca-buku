package dto

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=5"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=5"`
}

// OAuthRequest carries nothing but the Supabase access token the browser
// received. The identity behind it — email, name, avatar — is read back from
// Supabase rather than accepted from the client, so a caller cannot name
// whichever account it would like a token for.
type OAuthRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
