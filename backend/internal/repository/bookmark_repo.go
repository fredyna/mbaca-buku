package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fredy/mbaca-buku/internal/model"
)

type BookmarkRepository struct {
	db *sql.DB
}

func NewBookmarkRepository(db *sql.DB) *BookmarkRepository {
	return &BookmarkRepository{db: db}
}

func (r *BookmarkRepository) Create(ctx context.Context, bookmark *model.Bookmark) error {
	query := `INSERT INTO bookmarks (user_id, ebook_id, page_number, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query,
		bookmark.UserID, bookmark.EbookID, bookmark.PageNumber, bookmark.Note,
	).Scan(&bookmark.ID, &bookmark.CreatedAt)
}

func (r *BookmarkRepository) ListByUserAndEbook(ctx context.Context, userID, ebookID string) ([]model.Bookmark, error) {
	query := `SELECT id, user_id, ebook_id, page_number, note, created_at
		FROM bookmarks WHERE user_id = $1 AND ebook_id = $2
		ORDER BY page_number ASC`

	rows, err := r.db.QueryContext(ctx, query, userID, ebookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.EbookID, &b.PageNumber, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}

	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	return bookmarks, nil
}

func (r *BookmarkRepository) Delete(ctx context.Context, id, userID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM bookmarks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("bookmark not found")
	}
	return nil
}
