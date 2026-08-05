package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fredy/mbaca-buku/internal/model"
)

// Accepted values for EbookFilter.Visibility and EbookFilter.Sort.
const (
	VisibilityAll     = "all"
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"

	SortTitle  = "title"
	SortLatest = "latest"
)

// EbookFilter narrows a listing further than the visibility rules already do.
// Zero values mean "no extra narrowing", so the empty filter lists everything
// the user is allowed to see.
type EbookFilter struct {
	Query      string // keyword matched against the title
	Author     string // exact author match
	Visibility string // VisibilityAll, VisibilityPublic, or VisibilityPrivate
	Sort       string // SortTitle or SortLatest
}

// normalizeEbookFilter trims input and falls back to defaults for values it
// does not recognise. A stale bookmark should still render a page rather than
// fail the request.
func normalizeEbookFilter(f EbookFilter) EbookFilter {
	f.Query = strings.TrimSpace(f.Query)
	f.Author = strings.TrimSpace(f.Author)

	if f.Visibility != VisibilityPublic && f.Visibility != VisibilityPrivate {
		f.Visibility = VisibilityAll
	}
	if f.Sort != SortLatest {
		f.Sort = SortTitle
	}
	return f
}

// escapeLike neutralises the wildcards a user may type, so searching for "%"
// matches a literal percent sign instead of every row.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// buildEbookFilter turns the visibility rules plus the user's filter into a
// WHERE clause, an ORDER BY clause, and their arguments. Rows are addressed
// through the alias "e". It is pure so the generated SQL can be tested without
// a database.
//
// Visibility narrows within what the user may already see: a regular user
// asking for private ebooks gets their own, never someone else's.
func buildEbookFilter(f EbookFilter, userID string, isAdmin bool) (where, orderBy string, args []interface{}) {
	f = normalizeEbookFilter(f)

	var conds []string
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if !isAdmin {
		conds = append(conds, fmt.Sprintf("(e.is_private = false OR e.uploaded_by = %s)", arg(userID)))
	}

	switch f.Visibility {
	case VisibilityPublic:
		conds = append(conds, "e.is_private = false")
	case VisibilityPrivate:
		if isAdmin {
			conds = append(conds, "e.is_private = true")
		} else {
			conds = append(conds, fmt.Sprintf("(e.is_private = true AND e.uploaded_by = %s)", arg(userID)))
		}
	}

	if f.Query != "" {
		conds = append(conds, fmt.Sprintf(`e.title ILIKE %s ESCAPE '\'`, arg("%"+likeEscaper.Replace(f.Query)+"%")))
	}

	if f.Author != "" {
		conds = append(conds, fmt.Sprintf("e.author = %s", arg(f.Author)))
	}

	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// The id tie-breaker keeps paging stable when titles or timestamps collide;
	// without it a row can repeat on one page and vanish from the next.
	if f.Sort == SortLatest {
		orderBy = "ORDER BY e.created_at DESC, e.id DESC"
	} else {
		orderBy = "ORDER BY LOWER(e.title) ASC, e.id ASC"
	}

	return where, orderBy, args
}

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

const ebookColumns = `e.id, e.title, e.author, e.cover_url, e.file_url, e.file_size, e.total_pages, e.uploaded_by, COALESCE(u.name, ''), e.is_private, e.created_at, e.updated_at`

// The query assembly below is factored out of the repository methods so the
// finished SQL can be asserted in tests without a database.

func ebookCountQuery(where string) string {
	return `SELECT COUNT(*) FROM ebooks e ` + where
}

// argCount is the number of filter arguments already in use; the LIMIT and
// OFFSET placeholders continue from there.
func ebookListQuery(where, orderBy string, argCount int) string {
	return `SELECT ` + ebookColumns + ` FROM ebooks e LEFT JOIN users u ON e.uploaded_by = u.id ` +
		where + ` ` + orderBy + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argCount+1, argCount+2)
}

// ebookAuthorsQuery wraps the DISTINCT in a subquery because Postgres rejects
// an ORDER BY expression that is not part of a SELECT DISTINCT list.
func ebookAuthorsQuery(where string) string {
	if where == "" {
		where = `WHERE e.author <> ''`
	} else {
		where += ` AND e.author <> ''`
	}
	return `SELECT author FROM (SELECT DISTINCT e.author FROM ebooks e ` + where + `) t ORDER BY LOWER(author) ASC`
}

// ListVisible returns the page of ebooks the given user is allowed to see,
// narrowed by filter. Admins see everything; other users see public ebooks
// (is_private = false) plus their own private ones. The total count reflects
// the same filter, so the caller can derive the page count from it.
func (r *EbookRepository) ListVisible(ctx context.Context, page, perPage int, userID string, isAdmin bool, filter EbookFilter) ([]*model.Ebook, int, error) {
	where, orderBy, args := buildEbookFilter(filter, userID, isAdmin)

	var total int
	if err := r.db.QueryRowContext(ctx, ebookCountQuery(where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	listQuery := ebookListQuery(where, orderBy, len(args))
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

// ListAuthors returns the distinct authors across every ebook the user may
// see, ignoring any other filter so the dropdown it feeds stays stable while
// the user narrows the list. Ebooks without an author contribute no entry.
func (r *EbookRepository) ListAuthors(ctx context.Context, userID string, isAdmin bool) ([]string, error) {
	where, _, args := buildEbookFilter(EbookFilter{}, userID, isAdmin)

	rows, err := r.db.QueryContext(ctx, ebookAuthorsQuery(where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	authors := make([]string, 0)
	for rows.Next() {
		var author string
		if err := rows.Scan(&author); err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	return authors, rows.Err()
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
