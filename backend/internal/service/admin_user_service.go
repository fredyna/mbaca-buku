package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/dto"
	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

var (
	ErrCannotModifySelf = errors.New("cannot modify your own role or delete your own account")
	ErrLastAdmin        = errors.New("cannot remove the last remaining admin")
	ErrEmailTaken       = errors.New("email already registered")
)

type AdminUserService struct {
	userRepo *repository.UserRepository
}

func NewAdminUserService(userRepo *repository.UserRepository) *AdminUserService {
	return &AdminUserService{userRepo: userRepo}
}

func toAdminUserResponse(u *model.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (s *AdminUserService) List(ctx context.Context, page, perPage int) ([]dto.AdminUserResponse, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	users, total, err := s.userRepo.List(ctx, page, perPage)
	if err != nil {
		return nil, 0, err
	}

	out := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toAdminUserResponse(u))
	}
	return out, total, nil
}

func (s *AdminUserService) Create(ctx context.Context, req dto.AdminUserCreateRequest) (*dto.AdminUserResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	resp := toAdminUserResponse(user)
	return &resp, nil
}

func (s *AdminUserService) Update(ctx context.Context, id string, req dto.AdminUserUpdateRequest, actorID string) (*dto.AdminUserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if id == actorID && user.Role == "admin" && req.Role != "admin" {
		return nil, ErrCannotModifySelf
	}

	if user.Role == "admin" && req.Role != "admin" {
		count, err := s.userRepo.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, ErrLastAdmin
		}
	}

	if req.Email != user.Email {
		other, _ := s.userRepo.GetByEmail(ctx, req.Email)
		if other != nil && other.ID != user.ID {
			return nil, ErrEmailTaken
		}
	}

	user.Name = req.Name
	user.Email = req.Email
	user.Role = req.Role

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	updated, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := toAdminUserResponse(updated)
	return &resp, nil
}

func (s *AdminUserService) ResetPassword(ctx context.Context, id string, req dto.AdminPasswordResetRequest) error {
	if _, err := s.userRepo.GetByID(ctx, id); err != nil {
		return err
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	return s.userRepo.UpdatePassword(ctx, id, hash)
}

func (s *AdminUserService) Delete(ctx context.Context, id, actorID string) error {
	if id == actorID {
		return ErrCannotModifySelf
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if user.Role == "admin" {
		count, err := s.userRepo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrLastAdmin
		}
	}

	return s.userRepo.Delete(ctx, id)
}
