package repository

import (
	"strings"
	"testing"
)

func TestNormalizeEbookFilter(t *testing.T) {
	tests := []struct {
		name string
		in   EbookFilter
		want EbookFilter
	}{
		{
			name: "empty filter gets defaults",
			in:   EbookFilter{},
			want: EbookFilter{Visibility: VisibilityAll, Sort: SortTitle},
		},
		{
			name: "unknown values fall back instead of failing",
			in:   EbookFilter{Visibility: "secret", Sort: "author"},
			want: EbookFilter{Visibility: VisibilityAll, Sort: SortTitle},
		},
		{
			name: "known values survive",
			in:   EbookFilter{Visibility: VisibilityPrivate, Sort: SortLatest},
			want: EbookFilter{Visibility: VisibilityPrivate, Sort: SortLatest},
		},
		{
			name: "query and author are trimmed",
			in:   EbookFilter{Query: "  bumi  ", Author: " Tere Liye "},
			want: EbookFilter{Query: "bumi", Author: "Tere Liye", Visibility: VisibilityAll, Sort: SortTitle},
		},
		{
			name: "whitespace-only query counts as absent",
			in:   EbookFilter{Query: "   "},
			want: EbookFilter{Visibility: VisibilityAll, Sort: SortTitle},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeEbookFilter(tt.in); got != tt.want {
				t.Errorf("normalizeEbookFilter() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildEbookFilterWhere(t *testing.T) {
	tests := []struct {
		name       string
		filter     EbookFilter
		isAdmin    bool
		wantWhere  string
		wantArgs   []interface{}
		wantAbsent []string
	}{
		{
			name:      "admin without filters has no restriction",
			filter:    EbookFilter{},
			isAdmin:   true,
			wantWhere: "",
			wantArgs:  nil,
		},
		{
			name:      "regular user is limited to public plus own",
			filter:    EbookFilter{},
			wantWhere: "WHERE (e.is_private = false OR e.uploaded_by = $1)",
			wantArgs:  []interface{}{"user-1"},
		},
		{
			name:      "public visibility",
			filter:    EbookFilter{Visibility: VisibilityPublic},
			wantWhere: "WHERE (e.is_private = false OR e.uploaded_by = $1) AND e.is_private = false",
			wantArgs:  []interface{}{"user-1"},
		},
		{
			name:      "private visibility for a regular user stays scoped to their own",
			filter:    EbookFilter{Visibility: VisibilityPrivate},
			wantWhere: "WHERE (e.is_private = false OR e.uploaded_by = $1) AND (e.is_private = true AND e.uploaded_by = $2)",
			wantArgs:  []interface{}{"user-1", "user-1"},
		},
		{
			name:      "private visibility for an admin spans every owner",
			filter:    EbookFilter{Visibility: VisibilityPrivate},
			isAdmin:   true,
			wantWhere: "WHERE e.is_private = true",
			wantArgs:  nil,
		},
		{
			name:      "keyword search",
			filter:    EbookFilter{Query: "bumi"},
			isAdmin:   true,
			wantWhere: `WHERE e.title ILIKE $1 ESCAPE '\'`,
			wantArgs:  []interface{}{"%bumi%"},
		},
		{
			name:      "author is matched exactly",
			filter:    EbookFilter{Author: "Tere Liye"},
			isAdmin:   true,
			wantWhere: "WHERE e.author = $1",
			wantArgs:  []interface{}{"Tere Liye"},
		},
		{
			name:      "every filter combines with ascending placeholders",
			filter:    EbookFilter{Query: "laut", Author: "Tere Liye", Visibility: VisibilityPublic},
			wantWhere: `WHERE (e.is_private = false OR e.uploaded_by = $1) AND e.is_private = false AND e.title ILIKE $2 ESCAPE '\' AND e.author = $3`,
			wantArgs:  []interface{}{"user-1", "%laut%", "Tere Liye"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, _, args := buildEbookFilter(tt.filter, "user-1", tt.isAdmin)

			if where != tt.wantWhere {
				t.Errorf("where =\n  %q\nwant\n  %q", where, tt.wantWhere)
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tt.wantArgs)
			}
			for i := range args {
				if args[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %v, want %v", i, args[i], tt.wantArgs[i])
				}
			}
		})
	}
}

// A user typing "%" or "_" must search for those characters, not match every
// row, so the wildcards are escaped before they reach ILIKE.
func TestBuildEbookFilterEscapesLikeWildcards(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{query: "%", want: `%\%%`},
		{query: "_", want: `%\_%`},
		{query: `a\b`, want: `%a\\b%`},
		{query: "50%_off", want: `%50\%\_off%`},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, _, args := buildEbookFilter(EbookFilter{Query: tt.query}, "user-1", true)

			if len(args) != 1 {
				t.Fatalf("args = %v, want exactly one", args)
			}
			if args[0] != tt.want {
				t.Errorf("pattern = %q, want %q", args[0], tt.want)
			}
		})
	}
}

// Concatenated SQL is easy to break with a missing space or a placeholder that
// restarts at $1. These assert the finished statements, since the project has
// no database to run them against.
func TestEbookQueryAssembly(t *testing.T) {
	t.Run("count and list for an unfiltered admin", func(t *testing.T) {
		where, orderBy, args := buildEbookFilter(EbookFilter{}, "user-1", true)

		wantCount := "SELECT COUNT(*) FROM ebooks e "
		if got := ebookCountQuery(where); got != wantCount {
			t.Errorf("count query = %q, want %q", got, wantCount)
		}

		// No filter arguments, so paging takes $1 and $2.
		if got := ebookListQuery(where, orderBy, len(args)); !strings.HasSuffix(got, " LIMIT $1 OFFSET $2") {
			t.Errorf("list query = %q, want it to end with LIMIT $1 OFFSET $2", got)
		}
	})

	t.Run("paging placeholders follow the filter arguments", func(t *testing.T) {
		where, orderBy, args := buildEbookFilter(
			EbookFilter{Query: "laut", Author: "Tere Liye", Visibility: VisibilityPublic}, "user-1", false)

		if len(args) != 3 {
			t.Fatalf("args = %v, want 3", args)
		}
		if got := ebookListQuery(where, orderBy, len(args)); !strings.HasSuffix(got, " LIMIT $4 OFFSET $5") {
			t.Errorf("list query = %q, want it to end with LIMIT $4 OFFSET $5", got)
		}
	})

	t.Run("list query keeps the clauses separated", func(t *testing.T) {
		where, orderBy, args := buildEbookFilter(EbookFilter{Sort: SortLatest}, "user-1", false)
		got := ebookListQuery(where, orderBy, len(args))

		want := "SELECT " + ebookColumns +
			" FROM ebooks e LEFT JOIN users u ON e.uploaded_by = u.id " +
			"WHERE (e.is_private = false OR e.uploaded_by = $1) " +
			"ORDER BY e.created_at DESC, e.id DESC LIMIT $2 OFFSET $3"
		if got != want {
			t.Errorf("list query =\n  %q\nwant\n  %q", got, want)
		}
	})

	t.Run("authors query for an admin", func(t *testing.T) {
		where, _, _ := buildEbookFilter(EbookFilter{}, "user-1", true)

		want := `SELECT author FROM (SELECT DISTINCT e.author FROM ebooks e WHERE e.author <> '') t ORDER BY LOWER(author) ASC`
		if got := ebookAuthorsQuery(where); got != want {
			t.Errorf("authors query =\n  %q\nwant\n  %q", got, want)
		}
	})

	t.Run("authors query appends to an existing where clause", func(t *testing.T) {
		where, _, args := buildEbookFilter(EbookFilter{}, "user-1", false)

		want := `SELECT author FROM (SELECT DISTINCT e.author FROM ebooks e ` +
			`WHERE (e.is_private = false OR e.uploaded_by = $1) AND e.author <> '') t ORDER BY LOWER(author) ASC`
		if got := ebookAuthorsQuery(where); got != want {
			t.Errorf("authors query =\n  %q\nwant\n  %q", got, want)
		}
		if len(args) != 1 || args[0] != "user-1" {
			t.Errorf("args = %v, want [user-1]", args)
		}
	})
}

func TestBuildEbookFilterOrderBy(t *testing.T) {
	tests := []struct {
		name string
		sort string
		want string
	}{
		{name: "title is the default", sort: "", want: "ORDER BY LOWER(e.title) ASC, e.id ASC"},
		{name: "explicit title", sort: SortTitle, want: "ORDER BY LOWER(e.title) ASC, e.id ASC"},
		{name: "latest", sort: SortLatest, want: "ORDER BY e.created_at DESC, e.id DESC"},
		{name: "unknown sort falls back to title", sort: "size", want: "ORDER BY LOWER(e.title) ASC, e.id ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, orderBy, _ := buildEbookFilter(EbookFilter{Sort: tt.sort}, "user-1", true)

			if orderBy != tt.want {
				t.Errorf("orderBy = %q, want %q", orderBy, tt.want)
			}
			// Paging repeats or drops rows when the sort key has ties, so every
			// ordering needs the id tie-breaker.
			if !strings.Contains(orderBy, "e.id") {
				t.Errorf("orderBy %q is missing the id tie-breaker", orderBy)
			}
		})
	}
}
