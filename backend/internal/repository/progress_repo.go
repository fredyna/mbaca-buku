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

// ClampToTotalPages caps saved progress for an ebook at its real page count
// and marks anyone parked on the final page as finished. Needed after a page
// count correction, where stored progress can point past the last page.
func (r *ProgressRepository) ClampToTotalPages(ctx context.Context, ebookID string, totalPages int) (int64, error) {
	query := `UPDATE reading_progress
		SET last_page = LEAST(last_page, $1),
		    status = CASE WHEN LEAST(last_page, $1) >= $1 THEN 'completed' ELSE 'reading' END,
		    updated_at = NOW()
		WHERE ebook_id = $2`
	result, err := r.db.ExecContext(ctx, query, totalPages, ebookID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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
