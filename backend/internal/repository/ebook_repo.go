package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type EbookRepository struct {
	db *sql.DB
}

func NewEbookRepository(db *sql.DB) *EbookRepository {
	return &EbookRepository{db: db}
}

func (r *EbookRepository) Create(ctx context.Context, ebook *model.Ebook) error {
	query := `INSERT INTO ebooks (title, author, cover_url, file_url, file_size, total_pages, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		ebook.Title, ebook.Author, ebook.CoverURL, ebook.FileURL,
		ebook.FileSize, ebook.TotalPages, ebook.UploadedBy,
	).Scan(&ebook.ID, &ebook.CreatedAt, &ebook.UpdatedAt)
}

func (r *EbookRepository) GetByID(ctx context.Context, id string) (*model.Ebook, error) {
	ebook := &model.Ebook{}
	query := `SELECT id, title, author, cover_url, file_url, file_size, total_pages, uploaded_by, created_at, updated_at
		FROM ebooks WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ebook.ID, &ebook.Title, &ebook.Author, &ebook.CoverURL, &ebook.FileURL,
		&ebook.FileSize, &ebook.TotalPages, &ebook.UploadedBy, &ebook.CreatedAt, &ebook.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ebook not found")
	}
	return ebook, err
}

func (r *EbookRepository) List(ctx context.Context, page, perPage int) ([]*model.Ebook, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ebooks`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := `SELECT id, title, author, cover_url, file_url, file_size, total_pages, uploaded_by, created_at, updated_at
		FROM ebooks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var ebooks []*model.Ebook
	for rows.Next() {
		e := &model.Ebook{}
		if err := rows.Scan(&e.ID, &e.Title, &e.Author, &e.CoverURL, &e.FileURL,
			&e.FileSize, &e.TotalPages, &e.UploadedBy, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		ebooks = append(ebooks, e)
	}

	return ebooks, total, nil
}

func (r *EbookRepository) Update(ctx context.Context, ebook *model.Ebook) error {
	query := `UPDATE ebooks SET title = $1, author = $2, updated_at = NOW() WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, ebook.Title, ebook.Author, ebook.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ebook not found")
	}
	return nil
}

func (r *EbookRepository) Delete(ctx context.Context, id string) (*model.Ebook, error) {
	ebook, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, `DELETE FROM ebooks WHERE id = $1`, id)
	return ebook, err
}
