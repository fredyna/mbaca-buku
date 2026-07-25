package repository

import (
	"context"
	"database/sql"

	"github.com/fredy/mbaca-buku/internal/model"
)

type ProgressRepository struct {
	db *sql.DB
}

func NewProgressRepository(db *sql.DB) *ProgressRepository {
	return &ProgressRepository{db: db}
}

func (r *ProgressRepository) Upsert(ctx context.Context, userID, ebookID string, lastPage int, status string) error {
	query := `INSERT INTO reading_progress (user_id, ebook_id, last_page, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ebook_id)
		DO UPDATE SET last_page = $3, status = $4, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, userID, ebookID, lastPage, status)
	return err
}

func (r *ProgressRepository) GetByUserAndEbook(ctx context.Context, userID, ebookID string) (*model.ReadingProgress, error) {
	p := &model.ReadingProgress{}
	query := `SELECT id, user_id, ebook_id, last_page, status, updated_at
		FROM reading_progress WHERE user_id = $1 AND ebook_id = $2`
	err := r.db.QueryRowContext(ctx, query, userID, ebookID).
		Scan(&p.ID, &p.UserID, &p.EbookID, &p.LastPage, &p.Status, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}
