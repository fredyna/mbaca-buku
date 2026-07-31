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
	query := `INSERT INTO ebooks (title, author, cover_url, file_url, file_size, total_pages, uploaded_by, is_private)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowContext(ctx, query,
		ebook.Title, ebook.Author, ebook.CoverURL, ebook.FileURL,
		ebook.FileSize, ebook.TotalPages, ebook.UploadedBy, ebook.IsPrivate,
	).Scan(&ebook.ID, &ebook.CreatedAt, &ebook.UpdatedAt)
}

func (r *EbookRepository) GetByID(ctx context.Context, id string) (*model.Ebook, error) {
	ebook := &model.Ebook{}
	query := `SELECT e.id, e.title, e.author, e.cover_url, e.file_url, e.file_size, e.total_pages, e.uploaded_by, COALESCE(u.name, ''), e.is_private, e.created_at, e.updated_at
		FROM ebooks e LEFT JOIN users u ON e.uploaded_by = u.id WHERE e.id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ebook.ID, &ebook.Title, &ebook.Author, &ebook.CoverURL, &ebook.FileURL,
		&ebook.FileSize, &ebook.TotalPages, &ebook.UploadedBy, &ebook.UploadedByName, &ebook.IsPrivate,
		&ebook.CreatedAt, &ebook.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ebook not found")
	}
	return ebook, err
}

// ListVisible returns ebooks that the given user is allowed to see. Admins see
// everything; other users see public ebooks (is_private = false) plus their own
// private ones.
func (r *EbookRepository) ListVisible(ctx context.Context, page, perPage int, userID string, isAdmin bool) ([]*model.Ebook, int, error) {
	var (
		where string
		args  []interface{}
	)

	if isAdmin {
		where = ""
	} else {
		where = `WHERE is_private = false OR uploaded_by = $1`
		args = append(args, userID)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM ebooks ` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	listQuery := `SELECT e.id, e.title, e.author, e.cover_url, e.file_url, e.file_size, e.total_pages, e.uploaded_by, COALESCE(u.name, ''), e.is_private, e.created_at, e.updated_at
		FROM ebooks e LEFT JOIN users u ON e.uploaded_by = u.id ` + where + fmt.Sprintf(` ORDER BY e.created_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	ebooks := make([]*model.Ebook, 0)
	for rows.Next() {
		e := &model.Ebook{}
		if err := rows.Scan(&e.ID, &e.Title, &e.Author, &e.CoverURL, &e.FileURL,
			&e.FileSize, &e.TotalPages, &e.UploadedBy, &e.UploadedByName, &e.IsPrivate,
			&e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		ebooks = append(ebooks, e)
	}

	return ebooks, total, rows.Err()
}

func (r *EbookRepository) Update(ctx context.Context, ebook *model.Ebook) error {
	query := `UPDATE ebooks SET title = $1, author = $2, is_private = $3, updated_at = NOW() WHERE id = $4`
	result, err := r.db.ExecContext(ctx, query, ebook.Title, ebook.Author, ebook.IsPrivate, ebook.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ebook not found")
	}
	return nil
}

// ListAll returns every ebook, ignoring visibility rules. It is meant for
// maintenance tasks such as recomputing page counts, not for request handling.
func (r *EbookRepository) ListAll(ctx context.Context) ([]*model.Ebook, error) {
	query := `SELECT id, title, file_url, total_pages FROM ebooks ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ebooks := make([]*model.Ebook, 0)
	for rows.Next() {
		e := &model.Ebook{}
		if err := rows.Scan(&e.ID, &e.Title, &e.FileURL, &e.TotalPages); err != nil {
			return nil, err
		}
		ebooks = append(ebooks, e)
	}
	return ebooks, rows.Err()
}

// UpdateTotalPages corrects the stored page count for an ebook.
func (r *EbookRepository) UpdateTotalPages(ctx context.Context, id string, totalPages int) error {
	query := `UPDATE ebooks SET total_pages = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, totalPages, id)
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
