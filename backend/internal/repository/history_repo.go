package repository

import (
	"context"
	"database/sql"

	"github.com/fredy/mbaca-buku/internal/dto"
)

type HistoryRepository struct {
	db *sql.DB
}

func NewHistoryRepository(db *sql.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) LogOpen(ctx context.Context, userID, ebookID string) error {
	query := `INSERT INTO history (user_id, ebook_id) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, userID, ebookID)
	return err
}

func (r *HistoryRepository) GetUserHistory(ctx context.Context, userID string) ([]dto.HistoryItem, error) {
	query := `SELECT * FROM (
		SELECT DISTINCT ON (rp.ebook_id)
			rp.ebook_id, e.title, e.author, e.cover_url, e.total_pages, rp.last_page, rp.status,
			e.is_private, e.uploaded_by, COALESCE(u.name, ''), e.created_at, h.opened_at AS last_opened
		FROM reading_progress rp
		JOIN ebooks e ON e.id = rp.ebook_id
		LEFT JOIN users u ON u.id = e.uploaded_by
		LEFT JOIN history h ON h.user_id = rp.user_id AND h.ebook_id = rp.ebook_id
		WHERE rp.user_id = $1
		ORDER BY rp.ebook_id, h.opened_at DESC
	) AS history_items
	ORDER BY last_opened DESC NULLS LAST`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []dto.HistoryItem
	for rows.Next() {
		var item dto.HistoryItem
		var openedAt sql.NullTime
		var uploadedBy sql.NullString
		if err := rows.Scan(&item.EbookID, &item.Title, &item.Author, &item.CoverURL,
			&item.TotalPages, &item.LastPage, &item.Status,
			&item.IsPrivate, &uploadedBy, &item.UploadedByName, &item.CreatedAt, &openedAt); err != nil {
			return nil, err
		}
		if uploadedBy.Valid {
			item.UploadedBy = uploadedBy.String
		}
		if openedAt.Valid {
			item.LastOpened = openedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
