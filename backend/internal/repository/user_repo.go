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
	if user.Role == "" {
		user.Role = "user"
	}
	query := `INSERT INTO users (name, email, password_hash, role, provider, provider_id, avatar_url, is_oauth)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.PasswordHash, user.Role,
		user.Provider, user.ProviderID, user.AvatarURL, user.IsOAuth).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, role, provider, provider_id, avatar_url, is_oauth, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.Provider, &user.ProviderID, &user.AvatarURL, &user.IsOAuth, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, name, email, password_hash, role, provider, provider_id, avatar_url, is_oauth, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role, &user.Provider, &user.ProviderID, &user.AvatarURL, &user.IsOAuth, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return user, err
}

// List returns a page of users along with their latest activity.
//
// LEFT JOIN LATERAL is used because what's wanted is the latest *row* per user,
// not an aggregate: each subquery walks idx_activity_user_created /
// idx_activity_user_event backwards and stops at the first hit. LEFT keeps users
// who have never signed in in the list.
//
// The OS and browser come from the latest activity of any kind rather than from
// the latest login, because the question that column answers is "what are they
// using now".
func (r *UserRepository) List(ctx context.Context, page, perPage int) ([]*model.UserListItem, int, error) {
	offset := (page - 1) * perPage

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT u.id, u.name, u.email, u.role, u.created_at, u.updated_at,
		       last.created_at, last.os, last.browser,
		       login.created_at
		FROM users u
		LEFT JOIN LATERAL (
			SELECT created_at, os, browser FROM user_activity_logs
			WHERE user_id = u.id
			ORDER BY created_at DESC LIMIT 1
		) last ON TRUE
		LEFT JOIN LATERAL (
			SELECT created_at FROM user_activity_logs
			WHERE user_id = u.id AND event = $3
			ORDER BY created_at DESC LIMIT 1
		) login ON TRUE
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, perPage, offset, model.EventLogin)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]*model.UserListItem, 0)
	for rows.Next() {
		item := &model.UserListItem{}
		// os and browser are NOT NULL in the table, but the LEFT JOIN yields
		// NULL for a user with no activity at all, so they need Null scanners.
		var lastActive, lastLogin sql.NullTime
		var lastOS, lastBrowser sql.NullString

		if err := rows.Scan(
			&item.ID, &item.Name, &item.Email, &item.Role, &item.CreatedAt, &item.UpdatedAt,
			&lastActive, &lastOS, &lastBrowser, &lastLogin,
		); err != nil {
			return nil, 0, err
		}

		if lastActive.Valid {
			t := lastActive.Time
			item.LastActiveAt = &t
		}
		if lastLogin.Valid {
			t := lastLogin.Time
			item.LastLoginAt = &t
		}
		item.LastOS = lastOS.String
		item.LastBrowser = lastBrowser.String

		users = append(users, item)
	}
	return users, total, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET name = $1, email = $2, role = $3, provider = $4, provider_id = $5, avatar_url = $6, is_oauth = $7, updated_at = NOW() WHERE id = $8`,
		user.Name, user.Email, user.Role, user.Provider, user.ProviderID, user.AvatarURL, user.IsOAuth, user.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, hash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`, hash, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n)
	return n, err
}
