package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/supabase"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

var (
	ErrInvalidOldPassword = errors.New("old password is incorrect")
	ErrSamePassword       = errors.New("new password must be different from the old password")
	// ErrNoPassword reports a password login against an account that only ever
	// signed in through an identity provider, so it has no password to check.
	ErrNoPassword = errors.New("this account signs in with Google")
)

// UserStore is the subset of the user repository that AuthService depends on.
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	UpdatePassword(ctx context.Context, id, hash string) error
	Update(ctx context.Context, user *model.User) error
}

// OAuthVerifier resolves a third-party access token into the identity it was
// issued for. Implemented by pkg/supabase.Verifier.
type OAuthVerifier interface {
	GetUser(ctx context.Context, accessToken string) (*supabase.User, error)
	Configured() bool
}

type AuthService struct {
	userRepo  UserStore
	jwtSecret string
	oauth     OAuthVerifier
}

func NewAuthService(userRepo UserStore, jwtSecret string, oauth OAuthVerifier) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret, oauth: oauth}
}

// issueToken mints the app's own JWT for a user row. Every way of signing in —
// password, registration, Google — ends here, so the rest of the API only ever
// deals with one kind of token and one source of identity: the users table.
func (s *AuthService) issueToken(user *model.User) (*dto.AuthResponse, error) {
	token, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role},
		Token: token,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         "user",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.issueToken(user)
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// An account created through Google has no password hash. Saying so beats
	// "invalid email or password", which would send its owner off to reset a
	// password that never existed.
	if user.PasswordHash == "" {
		return nil, ErrNoPassword
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	return s.issueToken(user)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !utils.CheckPassword(user.PasswordHash, req.OldPassword) {
		return ErrInvalidOldPassword
	}

	if utils.CheckPassword(user.PasswordHash, req.NewPassword) {
		return ErrSamePassword
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.userRepo.UpdatePassword(ctx, userID, hash)
}

func (s *AuthService) SeedDefaultUser(ctx context.Context) error {
	existing, _ := s.userRepo.GetByEmail(ctx, "admin@mbacabuku.com")
	if existing != nil {
		return nil
	}

	hash, err := utils.HashPassword("12345")
	if err != nil {
		return err
	}

	user := &model.User{
		Name:         "admin",
		Email:        "admin@mbacabuku.com",
		PasswordHash: hash,
		Role:         "admin",
	}

	return s.userRepo.Create(ctx, user)
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// OAuthLogin redeems a Supabase access token, mirrors the identity behind it
// into the users table, and returns this API's own JWT for the resulting row.
//
// Accounts are keyed by email: signing in with Google using the address of an
// existing password account adopts that account rather than creating a second
// one, which keeps a person's reading history attached to them whichever button
// they pressed. The role always comes from the stored row, never from the
// provider, so an existing admin stays an admin and a new arrival is a user.
func (s *AuthService) OAuthLogin(ctx context.Context, req dto.OAuthRequest) (*dto.AuthResponse, error) {
	identity, err := s.oauth.GetUser(ctx, req.AccessToken)
	if err != nil {
		return nil, err
	}

	provider := identity.Provider
	if provider == "" {
		provider = "google"
	}

	user, _ := s.userRepo.GetByEmail(ctx, identity.Email)
	if user == nil {
		// No password hash: this account has no password until its owner sets
		// one, and CheckPassword rejects every candidate against an empty hash.
		user = &model.User{
			Name:       firstNonEmpty(identity.Name, identity.Email),
			Email:      identity.Email,
			Role:       "user",
			Provider:   provider,
			ProviderID: identity.ID,
			AvatarURL:  identity.AvatarURL,
			IsOAuth:    true,
		}
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		return s.issueToken(user)
	}

	updated := false
	if identity.Name != "" && identity.Name != user.Name {
		user.Name = identity.Name
		updated = true
	}
	if identity.AvatarURL != "" && identity.AvatarURL != user.AvatarURL {
		user.AvatarURL = identity.AvatarURL
		updated = true
	}
	if provider != user.Provider {
		user.Provider = provider
		updated = true
	}
	if identity.ID != "" && identity.ID != user.ProviderID {
		user.ProviderID = identity.ID
		updated = true
	}
	if !user.IsOAuth {
		user.IsOAuth = true
		updated = true
	}
	if updated {
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	return s.issueToken(user)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
