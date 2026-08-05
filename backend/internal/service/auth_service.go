package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

var (
	ErrInvalidOldPassword = errors.New("old password is incorrect")
	ErrSamePassword       = errors.New("new password must be different from the old password")
)

// UserStore is the subset of the user repository that AuthService depends on.
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id string) (*model.User, error)
	UpdatePassword(ctx context.Context, id, hash string) error
}

type AuthService struct {
	userRepo  UserStore
	jwtSecret string
}

func NewAuthService(userRepo UserStore, jwtSecret string) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSecret: jwtSecret}
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

	token, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role},
		Token: token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !utils.CheckPassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := utils.GenerateToken(user.ID, user.Role, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role},
		Token: token,
	}, nil
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
