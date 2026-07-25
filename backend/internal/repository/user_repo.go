package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.PasswordHash).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}
