package service

import (
	"context"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string) *AuthService {
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
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	token, err := utils.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email},
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

	token, err := utils.GenerateToken(user.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &dto.AuthResponse{
		User:  dto.UserResponse{ID: user.ID, Name: user.Name, Email: user.Email},
		Token: token,
	}, nil
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
	}

	return s.userRepo.Create(ctx, user)
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
