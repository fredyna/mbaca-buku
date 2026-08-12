# User Activity Log Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Record when each user last signed in, when they were last active, and which OS and browser they used, and show it in the admin users list.

**Architecture:** Every authenticated request passes through `AuthMiddleware`, which asks a Redis-backed throttle whether this user already has an `active` row inside the last 5 minutes; if not, a goroutine writes one row to a new `user_activity_logs` table. Sign-ins are recorded from `AuthHandler` instead, so `AuthService` stays free of HTTP concerns. The admin list reads the latest row per user with `LEFT JOIN LATERAL`, and an in-process daily job deletes rows older than 90 days.

**Tech Stack:** Go 1.25 + Gin, PostgreSQL (lib/pq), Redis (go-redis v9), React 19 + TypeScript + Tailwind 4, Vite.

Spec: `docs/superpowers/specs/2026-08-11-user-activity-log-design.md`

## Global Constraints

- **Migrations re-run on every startup.** `database.RunMigrations` (`backend/pkg/database/postgres.go:41`) globs and executes every `migrations/*.sql` on each boot with no version table. Every statement in a new migration must be idempotent (`IF NOT EXISTS`).
- **No NULL into Go strings.** lib/pq cannot scan `NULL` into `string`; migration 005 exists solely to fix that mistake. New text columns are `NOT NULL DEFAULT ''`. Where a `LEFT JOIN` can produce NULL, scan into `sql.NullString` / `sql.NullTime`.
- **No new Go dependencies.** The User-Agent parser is hand-written in `pkg/useragent`.
- **Activity logging must never fail a user's request.** Redis errors, insert errors, and parse misses are logged server-side and swallowed.
- **`*gin.Context` must not escape its request.** Copy request metadata into `model.RequestMeta` while the request is alive, before handing it to any goroutine.
- **Active window is 5 minutes**, used both as the throttle TTL and as the "is online" threshold in the UI, so an actively reading user never renders as offline.
- **Retention is 90 days.**
- **UI copy is English.** `UsersPage.tsx` already reads "Users", "Add User", "Joined", "Reset PW". The spec's mockup was written in Indonesian; the implementation uses English to match the existing page.
- **Backend test command:** `cd backend && go test ./...`
- **Frontend has no test runner** (no vitest/jest in `frontend/package.json`). Frontend tasks are verified with `npm run build` and `npm run lint`, plus a manual browser check.
- **Running SQL by hand.** The database is hosted (`DATABASE_URL` in `.env`) and `psql` is not installed on this machine. Use a throwaway container — the DB is reachable over the public internet, so no special Docker networking is needed:

  ```bash
  dbq() {
    docker run --rm postgres:16-alpine psql \
      "$(grep -m1 '^DATABASE_URL=' .env | cut -d= -f2- | tr -d '"')" -c "$1"
  }
  ```

  Then `dbq "SELECT 1"`. Run it from the repository root. The Supabase SQL editor works equally well for any of these checks.

## File Structure

| File | Responsibility |
|---|---|
| `backend/pkg/useragent/parse.go` | Pure: User-Agent string → browser/OS/device labels |
| `backend/pkg/useragent/parse_test.go` | Table-driven tests over real UA strings |
| `backend/pkg/cache/throttle.go` | Redis `SET NX EX` wrapper implementing one-per-window |
| `backend/pkg/utils/request.go` | `*gin.Context` → `model.RequestMeta` |
| `backend/migrations/006_add_user_activity_logs.sql` | Table + indexes |
| `backend/internal/model/activity.go` | `UserActivity`, `RequestMeta`, event constants |
| `backend/internal/model/user.go` | + `UserListItem` (user joined with latest activity) |
| `backend/internal/repository/activity_repo.go` | `Insert`, `DeleteOlderThan` |
| `backend/internal/repository/user_repo.go` | `List` gains the lateral-join activity columns |
| `backend/internal/service/activity_service.go` | Throttle, `Record`, `RecordAsync`, `StartCleanup` |
| `backend/internal/service/activity_service_test.go` | Throttle + failure-swallowing behaviour |
| `backend/internal/middleware/auth.go` | Records `active` after a token validates |
| `backend/internal/middleware/auth_test.go` | Recorder called once / not at all / nil-safe |
| `backend/internal/handler/auth_handler.go` | Records `login` on successful auth |
| `backend/internal/handler/auth_handler_test.go` | Login records, failed login does not |
| `backend/internal/dto/admin_user_dto.go` | + 4 activity fields on `AdminUserResponse` |
| `backend/internal/service/admin_user_service.go` | Maps `UserListItem` → response |
| `backend/internal/router/router.go` | Threads the recorder into `AuthMiddleware` |
| `backend/cmd/server/main.go` | Wires repo/throttle/service, starts cleanup |
| `frontend/src/utils/time.ts` | `formatRelativeTime`, `formatDateTime`, `isRecentlyActive` |
| `frontend/src/api/adminUsers.ts` | + 4 nullable fields on `AdminUser` |
| `frontend/src/pages/UsersPage.tsx` | Two new columns, desktop table + mobile cards |

---

### Task 1: User-Agent parser

The riskiest logic in the feature and the cheapest to test, so it goes first and stands alone.

**Files:**
- Create: `backend/pkg/useragent/parse.go`
- Test: `backend/pkg/useragent/parse_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `useragent.Parse(ua string) useragent.Info`, where `Info` has string fields `Browser`, `OS`, `Device`.

- [ ] **Step 1: Write the failing test**

Create `backend/pkg/useragent/parse_test.go`:

```go
package useragent

import "testing"

// Real User-Agent strings. The ordering traps are the point of this table:
// Edge and Opera both carry "Chrome" in their UA, Chrome carries "Safari",
// Android carries "Linux", and an iPad carries "Mac OS X".
func TestParse(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want Info
	}{
		{
			name: "Chrome on macOS",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "macOS", Device: "desktop"},
		},
		{
			name: "Edge is not reported as Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.2739.54",
			want: Info{Browser: "Edge 128", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Opera is not reported as Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 OPR/112.0.0.0",
			want: Info{Browser: "Opera 112", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Firefox on Windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0",
			want: Info{Browser: "Firefox 130", OS: "Windows", Device: "desktop"},
		},
		{
			name: "Safari on macOS is not reported as Chrome",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
			want: Info{Browser: "Safari 17", OS: "macOS", Device: "desktop"},
		},
		{
			name: "Chrome on an Android phone",
			ua:   "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "Android", Device: "mobile"},
		},
		{
			name: "Android without Mobile is a tablet",
			ua:   "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			want: Info{Browser: "Chrome 128", OS: "Android", Device: "tablet"},
		},
		{
			name: "Samsung Internet is not reported as Chrome",
			ua:   "Mozilla/5.0 (Linux; Android 14; SAMSUNG SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/26.0 Chrome/122.0.0.0 Mobile Safari/537.36",
			want: Info{Browser: "Samsung Internet 26", OS: "Android", Device: "mobile"},
		},
		{
			name: "Safari on iPhone is iOS, not macOS",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Safari 18", OS: "iOS", Device: "mobile"},
		},
		{
			name: "iPad is a tablet, not a desktop Mac",
			ua:   "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Safari 18", OS: "iOS", Device: "tablet"},
		},
		{
			name: "Chrome on iOS reports itself as CriOS",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0.6478.54 Mobile/15E148 Safari/604.1",
			want: Info{Browser: "Chrome 126", OS: "iOS", Device: "mobile"},
		},
		{
			name: "empty header yields empty labels rather than a guess",
			ua:   "",
			want: Info{},
		},
		{
			name: "unrecognised client yields empty labels rather than a guess",
			ua:   "curl/8.7.1",
			want: Info{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.ua)
			if got != tc.want {
				t.Errorf("Parse(%q)\n got: %+v\nwant: %+v", tc.ua, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./pkg/useragent/ -v`
Expected: FAIL — build error, `undefined: Parse` / `undefined: Info`.

- [ ] **Step 3: Write the implementation**

Create `backend/pkg/useragent/parse.go`:

```go
// Package useragent turns a browser's User-Agent header into the browser, OS
// and device labels shown in the admin activity log.
//
// The parsing is deliberately narrow: it recognises the clients this app
// actually sees and returns empty labels for anything else instead of guessing.
// Callers store the raw header alongside the parsed values, so an unrecognised
// device can still be identified by hand later.
package useragent

import (
	"regexp"
	"strings"
)

// Info is the parsed result. Every field may be empty when the header is
// missing or unrecognised.
type Info struct {
	Browser string // "Chrome 128"
	OS      string // "macOS"
	Device  string // "desktop" | "mobile" | "tablet"
}

// browserRules is scanned in order and the first match wins, because these
// tokens are nested by design: Edge, Opera and Samsung Internet all include
// "Chrome/" for compatibility, Chrome includes "Safari/", and the iOS builds of
// Chrome and Firefox announce themselves as CriOS and FxiOS while still
// including "Safari/". Reordering this list silently mislabels browsers.
var browserRules = []struct {
	name    string
	token   string
	version *regexp.Regexp
}{
	{"Edge", "Edg/", regexp.MustCompile(`Edg/(\d+)`)},
	{"Samsung Internet", "SamsungBrowser/", regexp.MustCompile(`SamsungBrowser/(\d+)`)},
	{"Opera", "OPR/", regexp.MustCompile(`OPR/(\d+)`)},
	{"Chrome", "CriOS/", regexp.MustCompile(`CriOS/(\d+)`)},
	{"Firefox", "FxiOS/", regexp.MustCompile(`FxiOS/(\d+)`)},
	{"Firefox", "Firefox/", regexp.MustCompile(`Firefox/(\d+)`)},
	{"Chrome", "Chrome/", regexp.MustCompile(`Chrome/(\d+)`)},
	{"Safari", "Safari/", regexp.MustCompile(`Version/(\d+)`)},
}

// osRules is likewise ordered: an Android UA contains "Linux", and an iPad UA
// contains "Mac OS X", so the specific platforms are tested before the generic
// ones they are built on.
var osRules = []struct {
	name  string
	token string
}{
	{"Windows", "Windows NT"},
	{"Android", "Android"},
	{"iOS", "iPhone"},
	{"iOS", "iPad"},
	{"iOS", "iPod"},
	{"macOS", "Macintosh"},
	{"Linux", "Linux"},
}

func Parse(ua string) Info {
	if strings.TrimSpace(ua) == "" {
		return Info{}
	}
	os := parseOS(ua)
	return Info{
		Browser: parseBrowser(ua),
		OS:      os,
		Device:  parseDevice(ua, os),
	}
}

func parseBrowser(ua string) string {
	for _, rule := range browserRules {
		if !strings.Contains(ua, rule.token) {
			continue
		}
		// A matching token with no parseable version still identifies the
		// browser, which is more useful than reporting nothing.
		if m := rule.version.FindStringSubmatch(ua); len(m) == 2 {
			return rule.name + " " + m[1]
		}
		return rule.name
	}
	return ""
}

func parseOS(ua string) string {
	for _, rule := range osRules {
		if strings.Contains(ua, rule.token) {
			return rule.name
		}
	}
	return ""
}

// parseDevice takes the already-resolved os so an unrecognised client is not
// reported as a desktop: "desktop" is a claim about the request, not a default.
func parseDevice(ua string, os string) string {
	switch {
	case strings.Contains(ua, "iPad"), strings.Contains(ua, "Tablet"):
		return "tablet"
	case strings.Contains(ua, "Android") && !strings.Contains(ua, "Mobile"):
		// Android tablets drop the "Mobile" token; phones keep it.
		return "tablet"
	case strings.Contains(ua, "Mobile"), strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPod"):
		return "mobile"
	case os != "":
		return "desktop"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd backend && go test ./pkg/useragent/ -v`
Expected: PASS — 13 subtests.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/useragent/
git commit -m "feat: add user-agent parser for activity logging"
```

---

### Task 2: Table, model and repository

**Files:**
- Create: `backend/migrations/006_add_user_activity_logs.sql`
- Create: `backend/internal/model/activity.go`
- Create: `backend/internal/repository/activity_repo.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `model.UserActivity` struct with fields `ID, UserID, Event, OS, Browser, Device, IPAddress, UserAgent string` and `CreatedAt time.Time`
  - `model.RequestMeta{IP, UserAgent string}`
  - `model.EventLogin = "login"`, `model.EventActive = "active"`
  - `repository.NewActivityRepository(db *sql.DB) *ActivityRepository`
  - `(*ActivityRepository).Insert(ctx context.Context, a *model.UserActivity) error`
  - `(*ActivityRepository).DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)`

There is no test in this task. `internal/repository` has no database test infrastructure — the one existing test there (`ebook_repo_test.go`) covers a pure helper function, not SQL. These two methods are verified end-to-end in Task 4's manual check.

- [ ] **Step 1: Write the migration**

Create `backend/migrations/006_add_user_activity_logs.sql`:

```sql
-- Every statement here is IF NOT EXISTS because RunMigrations re-executes every
-- file in this directory on each server start; there is no version table.
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event      VARCHAR(20) NOT NULL,              -- 'login' | 'active'
    -- Text columns are NOT NULL DEFAULT '' so lib/pq can scan them into plain
    -- Go strings; see migration 005 for what NULL here costs.
    os         VARCHAR(50)  NOT NULL DEFAULT '',
    browser    VARCHAR(50)  NOT NULL DEFAULT '',
    device     VARCHAR(20)  NOT NULL DEFAULT '',  -- desktop | mobile | tablet
    ip_address VARCHAR(45)  NOT NULL DEFAULT '',  -- 45 = longest possible IPv6 text form
    user_agent TEXT         NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Serves "latest activity for this user" in the admin list.
CREATE INDEX IF NOT EXISTS idx_activity_user_created
    ON user_activity_logs(user_id, created_at DESC);

-- Serves "latest login for this user" in the same query.
CREATE INDEX IF NOT EXISTS idx_activity_user_event
    ON user_activity_logs(user_id, event, created_at DESC);

-- Serves the retention sweep.
CREATE INDEX IF NOT EXISTS idx_activity_created
    ON user_activity_logs(created_at);
```

- [ ] **Step 2: Write the model**

Create `backend/internal/model/activity.go`:

```go
package model

import "time"

// Event values stored in user_activity_logs.event.
const (
	// EventLogin is one successful sign-in: password, registration or OAuth.
	EventLogin = "login"
	// EventActive is a throttled sign of life from an authenticated request.
	EventActive = "active"
)

// UserActivity is one row of the activity log.
type UserActivity struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Event     string    `json:"event"`
	OS        string    `json:"os"`
	Browser   string    `json:"browser"`
	Device    string    `json:"device"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// RequestMeta carries the only parts of an HTTP request the activity log needs.
// It exists so the service layer never depends on gin, and so the values are
// copied out while the request is still alive: activity rows are written from a
// goroutine that outlives the request, and *gin.Context is recycled by then.
type RequestMeta struct {
	IP        string
	UserAgent string
}
```

- [ ] **Step 3: Write the repository**

Create `backend/internal/repository/activity_repo.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/fredy/mbaca-buku/internal/model"
)

type ActivityRepository struct {
	db *sql.DB
}

func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Insert writes one activity row. id and created_at come from column defaults;
// nothing reads them back, so the round trip of a RETURNING clause is skipped.
func (r *ActivityRepository) Insert(ctx context.Context, a *model.UserActivity) error {
	query := `INSERT INTO user_activity_logs
		(user_id, event, os, browser, device, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		a.UserID, a.Event, a.OS, a.Browser, a.Device, a.IPAddress, a.UserAgent)
	return err
}

// DeleteOlderThan drops rows past the retention window and reports how many it
// removed, so the caller can log something meaningful instead of a bare "done".
// The cutoff is computed in Go rather than as a SQL INTERVAL literal, keeping
// the retention period a single Go constant.
func (r *ActivityRepository) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM user_activity_logs WHERE created_at < $1`, time.Now().Add(-age))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 4: Verify it builds and existing tests still pass**

Run: `cd backend && go build ./... && go vet ./... && go test ./...`
Expected: no build or vet output; all existing tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/migrations/006_add_user_activity_logs.sql backend/internal/model/activity.go backend/internal/repository/activity_repo.go
git commit -m "feat: add user_activity_logs table, model and repository"
```

---

### Task 3: Activity service and Redis throttle

**Files:**
- Create: `backend/pkg/cache/throttle.go`
- Create: `backend/internal/service/activity_service.go`
- Test: `backend/internal/service/activity_service_test.go`

**Interfaces:**
- Consumes: `model.UserActivity`, `model.RequestMeta`, `model.EventLogin`, `model.EventActive` (Task 2); `useragent.Parse` (Task 1).
- Produces:
  - `service.ActivityStore` interface: `Insert(ctx, *model.UserActivity) error`, `DeleteOlderThan(ctx, time.Duration) (int64, error)` — satisfied by `*repository.ActivityRepository`
  - `service.Throttler` interface: `Allow(ctx context.Context, key string, window time.Duration) bool`
  - `service.NewActivityService(repo ActivityStore, throttler Throttler) *ActivityService`
  - `(*ActivityService).Record(ctx context.Context, userID, event string, meta model.RequestMeta)` — no return value
  - `(*ActivityService).RecordAsync(userID, event string, meta model.RequestMeta)`
  - `(*ActivityService).StartCleanup(ctx context.Context)`
  - `cache.NewRedisThrottle(rdb *redis.Client) *RedisThrottle` with `Allow` as above

- [ ] **Step 1: Write the failing test**

Create `backend/internal/service/activity_service_test.go`:

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
)

// fakeActivityStore records what was written rather than which methods were
// called, so tests can assert on the parsed OS/browser landing in the row.
type fakeActivityStore struct {
	inserted  []*model.UserActivity
	insertErr error

	deletedAge   time.Duration
	deleteCalls  int
	deleteErr    error
	deletedCount int64
}

func (f *fakeActivityStore) Insert(ctx context.Context, a *model.UserActivity) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = append(f.inserted, a)
	return nil
}

func (f *fakeActivityStore) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	f.deleteCalls++
	f.deletedAge = age
	return f.deletedCount, f.deleteErr
}

// fakeThrottler answers from a queue so a test can spell out "first call
// allowed, second denied" without depending on a clock.
type fakeThrottler struct {
	answers []bool
	keys    []string
	window  time.Duration
}

func (f *fakeThrottler) Allow(ctx context.Context, key string, window time.Duration) bool {
	f.keys = append(f.keys, key)
	f.window = window
	if len(f.answers) == 0 {
		return true
	}
	answer := f.answers[0]
	f.answers = f.answers[1:]
	return answer
}

const chromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"

func TestRecordActiveWritesOncePerWindow(t *testing.T) {
	store := &fakeActivityStore{}
	throttler := &fakeThrottler{answers: []bool{true, false}}
	svc := NewActivityService(store, throttler)

	meta := model.RequestMeta{IP: "203.0.113.7", UserAgent: chromeMac}
	svc.Record(context.Background(), "user-1", model.EventActive, meta)
	svc.Record(context.Background(), "user-1", model.EventActive, meta)

	require.Len(t, store.inserted, 1, "second request inside the window must not write a row")
	assert.Equal(t, []string{"activity:seen:user-1", "activity:seen:user-1"}, throttler.keys)
	assert.Equal(t, 5*time.Minute, throttler.window)
}

func TestRecordActiveStoresParsedDeviceDetails(t *testing.T) {
	store := &fakeActivityStore{}
	svc := NewActivityService(store, &fakeThrottler{})

	svc.Record(context.Background(), "user-1", model.EventActive,
		model.RequestMeta{IP: "203.0.113.7", UserAgent: chromeMac})

	require.Len(t, store.inserted, 1)
	row := store.inserted[0]
	assert.Equal(t, "user-1", row.UserID)
	assert.Equal(t, model.EventActive, row.Event)
	assert.Equal(t, "Chrome 128", row.Browser)
	assert.Equal(t, "macOS", row.OS)
	assert.Equal(t, "desktop", row.Device)
	assert.Equal(t, "203.0.113.7", row.IPAddress)
	assert.Equal(t, chromeMac, row.UserAgent, "raw header is kept so a parse miss stays recoverable")
}

// Each sign-in is a distinct event worth seeing, and sign-ins are rare enough
// not to need rate limiting — so the throttle must not apply to them.
func TestRecordLoginIgnoresTheThrottle(t *testing.T) {
	store := &fakeActivityStore{}
	throttler := &fakeThrottler{answers: []bool{false, false}}
	svc := NewActivityService(store, throttler)

	meta := model.RequestMeta{IP: "203.0.113.7", UserAgent: chromeMac}
	svc.Record(context.Background(), "user-1", model.EventLogin, meta)
	svc.Record(context.Background(), "user-1", model.EventLogin, meta)

	assert.Len(t, store.inserted, 2)
	assert.Empty(t, throttler.keys, "login must not consult the throttle at all")
}

// Activity logging is bookkeeping attached to somebody else's request. A broken
// database or Redis must not surface to that user, so Record has no error to
// return and must not panic.
func TestRecordSwallowsStoreFailures(t *testing.T) {
	store := &fakeActivityStore{insertErr: errors.New("database is down")}
	svc := NewActivityService(store, &fakeThrottler{})

	assert.NotPanics(t, func() {
		svc.Record(context.Background(), "user-1", model.EventLogin,
			model.RequestMeta{IP: "203.0.113.7", UserAgent: chromeMac})
	})
}

func TestRecordSkipsAnonymousRequests(t *testing.T) {
	store := &fakeActivityStore{}
	svc := NewActivityService(store, &fakeThrottler{})

	svc.Record(context.Background(), "", model.EventActive,
		model.RequestMeta{IP: "203.0.113.7", UserAgent: chromeMac})

	assert.Empty(t, store.inserted)
}

// An unrecognised client still gets a row: the raw header is the point.
func TestRecordStoresUnparseableUserAgent(t *testing.T) {
	store := &fakeActivityStore{}
	svc := NewActivityService(store, &fakeThrottler{})

	svc.Record(context.Background(), "user-1", model.EventLogin,
		model.RequestMeta{IP: "203.0.113.7", UserAgent: "curl/8.7.1"})

	require.Len(t, store.inserted, 1)
	row := store.inserted[0]
	assert.Empty(t, row.Browser)
	assert.Empty(t, row.OS)
	assert.Equal(t, "curl/8.7.1", row.UserAgent)
}

func TestStartCleanupSweepsImmediatelyWithNinetyDayRetention(t *testing.T) {
	store := &fakeActivityStore{deletedCount: 3}
	svc := NewActivityService(store, &fakeThrottler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartCleanup(ctx)

	// StartCleanup sweeps once before waiting for the ticker; poll briefly
	// rather than sleeping a fixed span.
	deadline := time.Now().Add(2 * time.Second)
	for store.deleteCalls == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	require.Equal(t, 1, store.deleteCalls, "cleanup must run at startup, not only on the next tick")
	assert.Equal(t, 90*24*time.Hour, store.deletedAge)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/service/ -run TestRecord -v`
Expected: FAIL — build error, `undefined: NewActivityService`.

- [ ] **Step 3: Write the Redis throttle**

Create `backend/pkg/cache/throttle.go`:

```go
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisThrottle answers "may this key act now?" in a single SET NX EX round
// trip. Activity logging uses it so that an idle or busy user costs one Redis
// call rather than a database read.
type RedisThrottle struct {
	rdb *redis.Client
}

func NewRedisThrottle(rdb *redis.Client) *RedisThrottle {
	return &RedisThrottle{rdb: rdb}
}

// Allow reports whether the caller may act, claiming the window when it says
// yes. A Redis failure answers no: losing an activity row is harmless, while
// treating an outage as "allowed" would write a row on every single request.
func (t *RedisThrottle) Allow(ctx context.Context, key string, window time.Duration) bool {
	claimed, err := t.rdb.SetNX(ctx, key, 1, window).Result()
	if err != nil {
		return false
	}
	return claimed
}
```

- [ ] **Step 4: Write the activity service**

Create `backend/internal/service/activity_service.go`:

```go
package service

import (
	"context"
	"log"
	"time"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/useragent"
)

const (
	// activeWindow is how long one 'active' row stands for. The admin UI uses
	// the same span to decide whether a user counts as online, so somebody who
	// is actively reading never renders as offline.
	activeWindow = 5 * time.Minute

	activityRetention       = 90 * 24 * time.Hour
	activityCleanupInterval = 24 * time.Hour

	// activityWriteTimeout bounds a write started from a goroutine that has no
	// request context left to inherit a deadline from.
	activityWriteTimeout = 5 * time.Second
)

// ActivityStore is the subset of the activity repository this service needs.
type ActivityStore interface {
	Insert(ctx context.Context, a *model.UserActivity) error
	DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)
}

// Throttler rate-limits by key. Implemented by cache.RedisThrottle; declared
// here as an interface so the throttling behaviour can be tested without Redis.
type Throttler interface {
	Allow(ctx context.Context, key string, window time.Duration) bool
}

type ActivityService struct {
	repo      ActivityStore
	throttler Throttler
}

func NewActivityService(repo ActivityStore, throttler Throttler) *ActivityService {
	return &ActivityService{repo: repo, throttler: throttler}
}

// Record stores one activity row.
//
// 'active' events are throttled to one row per user per activeWindow, so an
// idle user costs nothing and a busy one costs about twelve rows an hour.
// 'login' events always land: each sign-in is a distinct thing worth seeing,
// and there are few of them.
//
// It returns nothing on purpose. This is bookkeeping attached to somebody
// else's request, and no failure here should be visible to them — problems go
// to the server log instead.
func (s *ActivityService) Record(ctx context.Context, userID, event string, meta model.RequestMeta) {
	if userID == "" {
		return
	}

	if event == model.EventActive && !s.throttler.Allow(ctx, "activity:seen:"+userID, activeWindow) {
		return
	}

	info := useragent.Parse(meta.UserAgent)
	err := s.repo.Insert(ctx, &model.UserActivity{
		UserID:    userID,
		Event:     event,
		OS:        info.OS,
		Browser:   info.Browser,
		Device:    info.Device,
		IPAddress: meta.IP,
		UserAgent: meta.UserAgent,
	})
	if err != nil {
		log.Printf("activity log: could not record %s for user %s: %v", event, userID, err)
	}
}

// RecordAsync records off the request's critical path, so a slow database never
// slows a page down.
//
// The caller must already have copied meta out of its *gin.Context, which does
// not outlive the request; the request's context is deliberately not inherited
// for the same reason — it is cancelled the moment the response is written.
// The goroutine count is bounded by the throttle for 'active' events and by the
// login rate for 'login' events.
func (s *ActivityService) RecordAsync(userID, event string, meta model.RequestMeta) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), activityWriteTimeout)
		defer cancel()
		s.Record(ctx, userID, event, meta)
	}()
}

// StartCleanup prunes logs past the retention window: once now, then daily
// until ctx is cancelled. Running it in-process means retention survives a
// container restart without any external scheduler, and follows the same shape
// as ReadingService.StartFlusher.
func (s *ActivityService) StartCleanup(ctx context.Context) {
	go func() {
		s.cleanup(ctx)

		ticker := time.NewTicker(activityCleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanup(ctx)
			}
		}
	}()
	log.Printf("Activity log cleanup started (%d day retention, daily sweep)",
		int(activityRetention.Hours()/24))
}

func (s *ActivityService) cleanup(ctx context.Context) {
	removed, err := s.repo.DeleteOlderThan(ctx, activityRetention)
	if err != nil {
		log.Printf("activity log: cleanup failed: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("activity log: removed %d rows older than %d days",
			removed, int(activityRetention.Hours()/24))
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/service/ -v -run "TestRecord|TestStartCleanup"`
Expected: PASS — 7 tests.

- [ ] **Step 6: Run the whole suite with the race detector**

Because `RecordAsync` and `StartCleanup` spawn goroutines, run:
`cd backend && go test -race ./...`
Expected: all PASS, no race warnings.

- [ ] **Step 7: Commit**

```bash
git add backend/pkg/cache/throttle.go backend/internal/service/activity_service.go backend/internal/service/activity_service_test.go
git commit -m "feat: add activity service with redis throttle and daily cleanup"
```

---

### Task 4: Record `active` from the auth middleware

At the end of this task activity logging works end to end for `active` events, so this is the first task with a manual verification step.

**Files:**
- Create: `backend/pkg/utils/request.go`
- Modify: `backend/internal/middleware/auth.go`
- Modify: `backend/internal/router/router.go:8-20` (RouterConfig), and the three `AuthMiddleware(...)` call sites
- Modify: `backend/cmd/server/main.go`
- Test: `backend/internal/middleware/auth_test.go`

**Interfaces:**
- Consumes: `model.RequestMeta`, `model.EventActive` (Task 2); `service.NewActivityService`, `RecordAsync`, `StartCleanup` (Task 3); `cache.NewRedisThrottle` (Task 3); `repository.NewActivityRepository` (Task 2).
- Produces:
  - `utils.RequestMetaOf(c *gin.Context) model.RequestMeta`
  - `middleware.ActivityRecorder` interface: `RecordAsync(userID, event string, meta model.RequestMeta)`
  - `middleware.AuthMiddleware(jwtSecret string, recorder ActivityRecorder) gin.HandlerFunc` — `recorder` may be nil
  - `router.RouterConfig.ActivityRecorder middleware.ActivityRecorder`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/middleware/auth_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// recorderSpy stands in for *service.ActivityService. It records synchronously,
// which keeps these tests free of sleeps: the middleware calls RecordAsync and
// the spy decides whether that means a goroutine.
type recorderSpy struct {
	calls []recordedCall
}

type recordedCall struct {
	userID string
	event  string
	meta   model.RequestMeta
}

func (s *recorderSpy) RecordAsync(userID, event string, meta model.RequestMeta) {
	s.calls = append(s.calls, recordedCall{userID: userID, event: event, meta: meta})
}

const testSecret = "test-secret"

func authTestRouter(recorder ActivityRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/ebooks", AuthMiddleware(testSecret, recorder), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuthMiddlewareRecordsActivityForValidToken(t *testing.T) {
	token, err := utils.GenerateToken("user-1", "user", testSecret)
	require.NoError(t, err)

	spy := &recorderSpy{}
	req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	w := httptest.NewRecorder()

	authTestRouter(spy).ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, spy.calls, 1)
	assert.Equal(t, "user-1", spy.calls[0].userID)
	assert.Equal(t, model.EventActive, spy.calls[0].event)
	assert.Contains(t, spy.calls[0].meta.UserAgent, "Chrome/128")
	assert.NotEmpty(t, spy.calls[0].meta.IP, "the row is useless without the caller's address")
}

// An unauthenticated request has no user to attribute activity to, and letting
// a rejected caller write rows would make the log forgeable.
func TestAuthMiddlewareRecordsNothingForRejectedRequests(t *testing.T) {
	cases := []struct {
		name   string
		header string
	}{
		{"no authorization header", ""},
		{"not a bearer token", "Basic dXNlcjpwYXNz"},
		{"signed with the wrong secret", "Bearer " + mustToken(t, "user-1", "other-secret")},
		{"garbage token", "Bearer not-a-jwt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &recorderSpy{}
			req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()

			authTestRouter(spy).ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Empty(t, spy.calls)
		})
	}
}

// A nil recorder disables logging so tests and any future entry point can build
// the middleware without wiring up a database and Redis.
func TestAuthMiddlewareWorksWithoutARecorder(t *testing.T) {
	token, err := utils.GenerateToken("user-1", "user", testSecret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/ebooks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		authTestRouter(nil).ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func mustToken(t *testing.T, userID, secret string) string {
	t.Helper()
	token, err := utils.GenerateToken(userID, "user", secret)
	require.NoError(t, err)
	return token
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/middleware/ -run TestAuthMiddleware -v`
Expected: FAIL — build error, `undefined: ActivityRecorder` and "too many arguments in call to AuthMiddleware".

- [ ] **Step 3: Add the request-metadata helper**

Create `backend/pkg/utils/request.go`:

```go
package utils

import (
	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/model"
)

// RequestMetaOf copies the request details the activity log stores.
//
// Call it while the request is still being served: *gin.Context is recycled
// once the handler returns, so these values have to be taken out before any
// goroutine receives them. ClientIP reads X-Forwarded-For, which nginx sets in
// front of this service, so it yields the caller's address rather than the
// proxy's.
func RequestMetaOf(c *gin.Context) model.RequestMeta {
	return model.RequestMeta{
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
}
```

- [ ] **Step 4: Record activity in the middleware**

In `backend/internal/middleware/auth.go`, add the `model` import and replace the function with:

```go
// ActivityRecorder records a sign of life for an authenticated request.
// Satisfied by *service.ActivityService. A nil recorder disables recording,
// which keeps this middleware constructible without a database or Redis.
type ActivityRecorder interface {
	RecordAsync(userID, event string, meta model.RequestMeta)
}

func AuthMiddleware(jwtSecret string, recorder ActivityRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization format")
			c.Abort()
			return
		}

		userID, role, err := utils.ParseToken(parts[1], jwtSecret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("role", role)

		// Only past this point is there a user to attribute activity to. The
		// recorder throttles internally, so this fires on every request but
		// writes at most one row per user per window. Metadata is read here,
		// while the context is still valid, not inside the recorder's goroutine.
		if recorder != nil {
			recorder.RecordAsync(userID, model.EventActive, utils.RequestMetaOf(c))
		}

		c.Next()
	}
}
```

The import block becomes:

```go
import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/pkg/utils"
)
```

- [ ] **Step 5: Run the middleware tests to verify they pass**

Run: `cd backend && go test ./internal/middleware/ -v`
Expected: PASS — the three new tests plus the existing CORS test. The build of `./internal/router` still fails at this point; that is fixed in the next step.

- [ ] **Step 6: Thread the recorder through the router**

In `backend/internal/router/router.go`, add the field to `RouterConfig`:

```go
type RouterConfig struct {
	AuthHandler      *handler.AuthHandler
	EbookHandler     *handler.EbookHandler
	ReadingHandler   *handler.ReadingHandler
	HistoryHandler   *handler.HistoryHandler
	BookmarkHandler  *handler.BookmarkHandler
	AdminUserHandler *handler.AdminUserHandler
	ActivityRecorder middleware.ActivityRecorder
	JWTSecret        string
	AllowedOrigins   []string
}
```

Then update all three `AuthMiddleware` call sites to pass it:

```go
	auth.GET("/me", middleware.AuthMiddleware(cfg.JWTSecret, cfg.ActivityRecorder), cfg.AuthHandler.Me)
	auth.PUT("/password", middleware.AuthMiddleware(cfg.JWTSecret, cfg.ActivityRecorder), cfg.AuthHandler.ChangePassword)
```

```go
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret, cfg.ActivityRecorder))
```

- [ ] **Step 7: Wire it up in main.go**

In `backend/cmd/server/main.go`, add these two lines immediately **after** the `authService := service.NewAuthService(...)` block and **before** `authHandler := handler.NewAuthHandler(...)`. Task 5 changes that `NewAuthHandler` call to take `activityService`, so putting it here now avoids moving it later:

```go
	activityRepo := repository.NewActivityRepository(db)
	activityService := service.NewActivityService(activityRepo, cache.NewRedisThrottle(rdb))
```

Add `ActivityRecorder: activityService,` to the `router.Setup` config literal.

Start the cleanup alongside the existing flusher, reusing its context so both stop on shutdown:

```go
	flusherCtx, cancelFlusher := context.WithCancel(context.Background())
	defer cancelFlusher()
	readingService.StartFlusher(flusherCtx)
	activityService.StartCleanup(flusherCtx)
```

`pkg/cache` is already imported for `cache.NewRedisClient`, so no import changes are needed.

- [ ] **Step 8: Verify the whole backend builds and passes**

Run: `cd backend && go build ./... && go vet ./... && go test -race ./...`
Expected: no build or vet output; all tests PASS with no race warnings.

- [ ] **Step 9: Verify end to end against the running stack**

Run: `docker compose up -d --build backend`

Then confirm the migration applied and rows appear. Log in through the UI at `http://localhost:6900`, open a couple of pages, then check the table:

```bash
docker compose logs backend | grep -i "Migration applied: 006\|Activity log cleanup started"
```

Expected: both lines present.

```bash
dbq "SELECT user_id, event, os, browser, device, ip_address, created_at
     FROM user_activity_logs ORDER BY created_at DESC LIMIT 10"
```

Expected: at least one `active` row with a populated `os` and `browser`.

Then confirm the throttle: note the count, browse the app for a minute clicking through several pages, and check it again.

```bash
dbq "SELECT count(*) FROM user_activity_logs WHERE event = 'active'"
```

Expected: the count grew by at most one — refreshing repeatedly must **not** add a row per request.

- [ ] **Step 10: Commit**

```bash
git add backend/pkg/utils/request.go backend/internal/middleware/auth.go backend/internal/middleware/auth_test.go backend/internal/router/router.go backend/cmd/server/main.go
git commit -m "feat: record user activity from the auth middleware"
```

---

### Task 5: Record `login` from the auth handler

**Files:**
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `backend/cmd/server/main.go` (the `NewAuthHandler` call)
- Test: `backend/internal/handler/auth_handler_test.go`

**Interfaces:**
- Consumes: `model.RequestMeta`, `model.EventLogin` (Task 2); `utils.RequestMetaOf` (Task 4); `*service.ActivityService` (Task 3).
- Produces: `handler.NewAuthHandler(authService *service.AuthService, recorder activityRecorder) *AuthHandler` — note the **second parameter is new** and may be nil.

- [ ] **Step 1: Write the failing test**

Create `backend/internal/handler/auth_handler_test.go`:

```go
package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
	"github.com/fredy/mbaca-buku/internal/service"
	"github.com/fredy/mbaca-buku/pkg/utils"
)

// stubUserStore implements service.UserStore with a single known account, so
// these tests exercise the real AuthService rather than a fake of it — the
// point under test is what the handler does after a genuine sign-in succeeds.
type stubUserStore struct {
	user *model.User
}

func (s *stubUserStore) Create(ctx context.Context, user *model.User) error {
	user.ID = "created-id"
	return nil
}

func (s *stubUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if s.user != nil && s.user.Email == email {
		return s.user, nil
	}
	return nil, errors.New("user not found")
}

func (s *stubUserStore) GetByID(ctx context.Context, id string) (*model.User, error) {
	return s.user, nil
}

func (s *stubUserStore) UpdatePassword(ctx context.Context, id, hash string) error { return nil }

func (s *stubUserStore) Update(ctx context.Context, user *model.User) error { return nil }

type recorderSpy struct {
	calls []recordedCall
}

type recordedCall struct {
	userID string
	event  string
	meta   model.RequestMeta
}

func (s *recorderSpy) RecordAsync(userID, event string, meta model.RequestMeta) {
	s.calls = append(s.calls, recordedCall{userID: userID, event: event, meta: meta})
}

func loginTestRouter(t *testing.T, spy *recorderSpy) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	hash, err := utils.HashPassword("secret123")
	require.NoError(t, err)

	store := &stubUserStore{user: &model.User{
		ID:           "user-1",
		Name:         "Budi",
		Email:        "budi@example.com",
		PasswordHash: hash,
		Role:         "user",
	}}
	authService := service.NewAuthService(store, "test-secret", nil)
	h := NewAuthHandler(authService, spy)

	r := gin.New()
	r.POST("/api/auth/login", h.Login)
	return r
}

func postLogin(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLoginRecordsALoginEvent(t *testing.T) {
	spy := &recorderSpy{}
	w := postLogin(loginTestRouter(t, spy), `{"email":"budi@example.com","password":"secret123"}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, spy.calls, 1)
	assert.Equal(t, "user-1", spy.calls[0].userID)
	assert.Equal(t, model.EventLogin, spy.calls[0].event)
	assert.Contains(t, spy.calls[0].meta.UserAgent, "Firefox/130")
}

// A failed sign-in belongs in neither the activity log nor a "last login"
// column: it would show an account as active that nobody got into.
func TestLoginRecordsNothingWhenCredentialsAreWrong(t *testing.T) {
	spy := &recorderSpy{}
	w := postLogin(loginTestRouter(t, spy), `{"email":"budi@example.com","password":"wrong-password"}`)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, spy.calls)
}

func TestLoginWorksWithoutARecorder(t *testing.T) {
	w := postLogin(loginTestRouter(t, nil), `{"email":"budi@example.com","password":"secret123"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}
```

Note: `loginTestRouter(t, nil)` passes a typed nil `*recorderSpy` through the `activityRecorder` parameter. The handler's nil guard must therefore compare the interface value, which is why `NewAuthHandler` stores nil explicitly — see Step 3.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && go test ./internal/handler/ -run TestLogin -v`
Expected: FAIL — build error, "too many arguments in call to NewAuthHandler".

- [ ] **Step 3: Record logins in the handler**

In `backend/internal/handler/auth_handler.go`, replace the type and constructor:

```go
// activityRecorder records a sign-in for the account that just authenticated.
// Satisfied by *service.ActivityService.
//
// Sign-ins are recorded here rather than in AuthService so that service stays
// free of HTTP concerns: the User-Agent and client address only exist at this
// layer, and AuthService's tests keep working untouched.
type activityRecorder interface {
	RecordAsync(userID, event string, meta model.RequestMeta)
}

type AuthHandler struct {
	authService *service.AuthService
	recorder    activityRecorder
}

func NewAuthHandler(authService *service.AuthService, recorder activityRecorder) *AuthHandler {
	return &AuthHandler{authService: authService, recorder: recorder}
}

// recordLogin logs a successful sign-in. A nil recorder disables it, and a
// typed-nil pointer reaches the same check, so tests can pass either.
func (h *AuthHandler) recordLogin(c *gin.Context, userID string) {
	if h.recorder == nil {
		return
	}
	h.recorder.RecordAsync(userID, model.EventLogin, utils.RequestMetaOf(c))
}
```

Add `"github.com/fredy/mbaca-buku/internal/model"` to the import block.

Then add the call on each success path, immediately before the existing success response:

In `Register`:

```go
	h.recordLogin(c, resp.User.ID)
	utils.SuccessResponse(c, http.StatusCreated, resp)
```

In `Login`:

```go
	h.recordLogin(c, resp.User.ID)
	utils.SuccessResponse(c, http.StatusOK, resp)
```

In `OAuth`, inside the `case err == nil:` branch:

```go
	case err == nil:
		h.recordLogin(c, resp.User.ID)
		utils.SuccessResponse(c, http.StatusOK, resp)
```

Registration counts as a sign-in because `Register` returns a token — that user's session starts right there.

`Me` and `ChangePassword` are left alone: they sit behind `AuthMiddleware`, which already records them as activity.

- [ ] **Step 4: Update the call in main.go**

In `backend/cmd/server/main.go`, pass the `activityService` created in Task 4 into the handler:

```go
	authHandler := handler.NewAuthHandler(authService, activityService)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd backend && go test ./internal/handler/ -v -run TestLogin`
Expected: PASS — 3 tests.

- [ ] **Step 6: Verify the whole backend**

Run: `cd backend && go build ./... && go vet ./... && go test -race ./...`
Expected: no build or vet output; all PASS, no races.

- [ ] **Step 7: Verify end to end**

Run: `docker compose up -d --build backend`, sign out and sign in again at `http://localhost:6900`, then:

```bash
dbq "SELECT event, os, browser, device, created_at
     FROM user_activity_logs ORDER BY created_at DESC LIMIT 5"
```

Expected: a `login` row at the top with populated `os` and `browser`. Then enter a wrong password once and confirm no new `login` row appears.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/handler/auth_handler.go backend/internal/handler/auth_handler_test.go backend/cmd/server/main.go
git commit -m "feat: record login events for password, oauth and registration"
```

---

### Task 6: Expose last login and last active in the admin API

**Files:**
- Modify: `backend/internal/model/user.go`
- Modify: `backend/internal/repository/user_repo.go:52-79` (`List`)
- Modify: `backend/internal/dto/admin_user_dto.go:5-12` (`AdminUserResponse`)
- Modify: `backend/internal/service/admin_user_service.go:30-58`

**Interfaces:**
- Consumes: `model.EventLogin` (Task 2), the `user_activity_logs` table (Task 2).
- Produces:
  - `model.UserListItem` — embeds `User`, adds `LastLoginAt *time.Time`, `LastActiveAt *time.Time`, `LastOS string`, `LastBrowser string`
  - `(*UserRepository).List(ctx, page, perPage int) ([]*model.UserListItem, int, error)` — **return type changed** from `[]*model.User`
  - `dto.AdminUserResponse` gains `last_login_at`, `last_active_at`, `last_os`, `last_browser`

No unit test: this task is a SQL query plus straight field mapping, and `internal/repository` has no database test harness. It is verified against the running stack in Step 5.

- [ ] **Step 1: Add the list item model**

Append to `backend/internal/model/user.go`:

```go
// UserListItem is a users row joined with its most recent activity, as the admin
// user list needs it. The timestamps are pointers because a user who has never
// signed in has neither, and rendering that as the zero time would read as
// "January 1st, year 1" in the UI.
type UserListItem struct {
	User
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	LastOS       string     `json:"last_os"`
	LastBrowser  string     `json:"last_browser"`
}
```

- [ ] **Step 2: Rewrite the repository query**

Replace `List` in `backend/internal/repository/user_repo.go`:

```go
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
```

- [ ] **Step 3: Extend the DTO**

Replace `AdminUserResponse` in `backend/internal/dto/admin_user_dto.go`:

```go
type AdminUserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Activity fields, populated by the list endpoint only. Null means the user
	// has never signed in, which the UI shows differently from "a long time
	// ago"; Create and Update leave them null because a user cannot have
	// activity before they exist.
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	LastOS       string     `json:"last_os"`
	LastBrowser  string     `json:"last_browser"`
}
```

- [ ] **Step 4: Map it in the service**

In `backend/internal/service/admin_user_service.go`, add a second mapper below the existing `toAdminUserResponse` (which stays as-is for `Create` and `Update`):

```go
func toAdminUserListResponse(u *model.UserListItem) dto.AdminUserResponse {
	resp := toAdminUserResponse(&u.User)
	resp.LastLoginAt = u.LastLoginAt
	resp.LastActiveAt = u.LastActiveAt
	resp.LastOS = u.LastOS
	resp.LastBrowser = u.LastBrowser
	return resp
}
```

Then change the loop in `List` to use it:

```go
	out := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toAdminUserListResponse(u))
	}
	return out, total, nil
```

- [ ] **Step 5: Verify the backend and the response shape**

Run: `cd backend && go build ./... && go vet ./... && go test -race ./...`
Expected: no build or vet output; all PASS.

Then run `docker compose up -d --build backend`, sign in as an admin at `http://localhost:6900`, and check the endpoint:

```bash
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:6900/api/admin/users?page=1&per_page=20" | python3 -m json.tool
```

Expected: each user object carries `last_login_at`, `last_active_at`, `last_os`, `last_browser`. The signed-in admin has non-null timestamps and a populated browser; a user who has never signed in has `null` for both timestamps and `""` for the strings. `created_at` ordering is unchanged.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/model/user.go backend/internal/repository/user_repo.go backend/internal/dto/admin_user_dto.go backend/internal/service/admin_user_service.go
git commit -m "feat: return last login and last active in the admin user list"
```

---

### Task 7: Show the columns in the admin users page

**Files:**
- Create: `frontend/src/utils/time.ts`
- Modify: `frontend/src/api/adminUsers.ts:3-10` (`AdminUser`)
- Modify: `frontend/src/pages/UsersPage.tsx`

**Interfaces:**
- Consumes: the four new fields from Task 6.
- Produces: `formatRelativeTime`, `formatDateTime`, `isRecentlyActive` from `frontend/src/utils/time.ts`.

There is no test runner in this project, so verification is `npm run build`, `npm run lint`, and a browser check.

- [ ] **Step 1: Add the time helpers**

Create `frontend/src/utils/time.ts`:

```ts
/**
 * Timestamp formatting for the admin activity columns.
 *
 * Relative wording answers "is this person around?" at a glance, which an
 * absolute date does not; absolute formatting is kept for the sign-in column,
 * where the exact moment is the useful part.
 */

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/**
 * How recent counts as online. Matches the activity throttle window on the
 * server: a row is written at most once every five minutes, so a shorter
 * threshold here would show an actively reading user as offline.
 */
export const ACTIVE_WINDOW_MS = 5 * MINUTE;

export function formatRelativeTime(iso: string | null): string {
  if (!iso) return 'Never';

  const diff = Date.now() - new Date(iso).getTime();
  // A negative difference means the server clock is ahead of the browser's.
  // "Just now" is honest there; "in 3 minutes" would look broken.
  if (diff < MINUTE) return 'Just now';
  if (diff < HOUR) return `${Math.floor(diff / MINUTE)} min ago`;

  if (diff < DAY) {
    const hours = Math.floor(diff / HOUR);
    return `${hours} hour${hours === 1 ? '' : 's'} ago`;
  }

  const days = Math.floor(diff / DAY);
  // Past a month, "47 days ago" is harder to read than the date itself.
  if (days < 30) return `${days} day${days === 1 ? '' : 's'} ago`;
  return formatDateTime(iso);
}

export function formatDateTime(iso: string | null): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleString(undefined, {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function isRecentlyActive(iso: string | null): boolean {
  if (!iso) return false;
  return Date.now() - new Date(iso).getTime() < ACTIVE_WINDOW_MS;
}
```

- [ ] **Step 2: Extend the API type**

In `frontend/src/api/adminUsers.ts`, extend `AdminUser`:

```ts
export interface AdminUser {
  id: string;
  name: string;
  email: string;
  role: 'user' | 'admin';
  created_at: string;
  updated_at: string;
  // Null until the user first signs in. Create and update responses omit these,
  // so they are optional as well as nullable.
  last_login_at?: string | null;
  last_active_at?: string | null;
  last_os?: string;
  last_browser?: string;
}
```

- [ ] **Step 3: Add a shared activity cell component**

The desktop table and the mobile cards must not drift apart, so the "last active" rendering lives in one place. Add this to `frontend/src/pages/UsersPage.tsx`, above `export default function UsersPage()`:

```tsx
/** Last activity plus the device it came from, shared by the table and the cards. */
function LastActiveCell({ user }: { user: AdminUser }) {
  const lastActive = user.last_active_at ?? null;
  const online = isRecentlyActive(lastActive);
  const device = [user.last_browser, user.last_os].filter(Boolean).join(' · ');

  return (
    <div>
      <div className="flex items-center gap-1.5">
        {online && (
          <span
            className="w-2 h-2 rounded-full bg-green-500 shrink-0"
            title="Active in the last 5 minutes"
          />
        )}
        <span className={online ? 'text-gray-900' : 'text-gray-500'}>
          {lastActive ? formatRelativeTime(lastActive) : 'Never signed in'}
        </span>
      </div>
      {device && <div className="text-xs text-gray-400 mt-0.5">{device}</div>}
    </div>
  );
}
```

Add the import:

```ts
import { formatDateTime, formatRelativeTime, isRecentlyActive } from '../utils/time';
```

- [ ] **Step 4: Rework the desktop table**

Email and "Joined" move into the name cell so the table stays at five columns instead of growing to seven narrow ones. Replace the `<thead>` and the `<tbody>` row markup:

```tsx
              <thead className="bg-gray-50 text-gray-600 text-left">
                <tr>
                  <th className="px-4 py-3 font-medium">User</th>
                  <th className="px-4 py-3 font-medium">Role</th>
                  <th className="px-4 py-3 font-medium">Last login</th>
                  <th className="px-4 py-3 font-medium">Last active</th>
                  <th className="px-4 py-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {users.map((u) => {
                  const isSelf = u.id === currentUser?.id;
                  return (
                    <tr key={u.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <div className="text-gray-900">
                          {u.name}
                          {isSelf && <span className="ml-2 text-xs text-gray-400">(you)</span>}
                        </div>
                        <div className="text-gray-500">{u.email}</div>
                        <div className="text-xs text-gray-400 mt-0.5">
                          Joined {new Date(u.created_at).toLocaleDateString()}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-block px-2 py-0.5 text-xs rounded ${
                            u.role === 'admin'
                              ? 'bg-purple-100 text-purple-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {u.role}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
                        {u.last_login_at ? formatDateTime(u.last_login_at) : '—'}
                      </td>
                      <td className="px-4 py-3">
                        <LastActiveCell user={u} />
                      </td>
                      <td className="px-4 py-3 text-right align-top">
                        <div className="inline-flex gap-2">
                          <button
                            onClick={() => setEditing(u)}
                            className="text-blue-600 hover:text-blue-800"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => setResettingPassword(u)}
                            className="text-gray-600 hover:text-gray-900"
                          >
                            Reset PW
                          </button>
                          <button
                            onClick={() => handleDelete(u)}
                            disabled={isSelf}
                            className="text-red-600 hover:text-red-800 disabled:opacity-40 disabled:cursor-not-allowed"
                            title={isSelf ? 'You cannot delete your own account' : ''}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
```

- [ ] **Step 5: Add the same information to the mobile cards**

In the `md:hidden` block, replace the single "Joined" line with the activity block:

```tsx
                  <div className="text-xs text-gray-400 mb-1">
                    Joined {new Date(u.created_at).toLocaleDateString()}
                  </div>
                  <div className="text-sm text-gray-500 mb-1">
                    Last login: {u.last_login_at ? formatDateTime(u.last_login_at) : '—'}
                  </div>
                  <div className="text-sm mb-3">
                    <LastActiveCell user={u} />
                  </div>
```

- [ ] **Step 6: Verify the build and the lint**

Run: `cd frontend && npm run lint && npm run build`
Expected: no lint findings and a successful `tsc -b` + `vite build`.

- [ ] **Step 7: Verify in the browser**

Run: `docker compose up -d --build`, then open `http://localhost:6900`, sign in as admin, and go to Users.

Expected:
- Your own row shows a green dot with "Just now" and your real browser and OS (e.g. "Chrome 128 · macOS").
- "Last login" shows today's date and time for your account.
- A user who has never signed in shows "—" under Last login and "Never signed in" under Last active, with no green dot and no device line.
- Narrow the window below `md`: the cards show the same three lines, and the table is gone.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/utils/time.ts frontend/src/api/adminUsers.ts frontend/src/pages/UsersPage.tsx
git commit -m "feat: show last login and last active in the admin users page"
```

---

## Verification Summary

After Task 7, the full check:

```bash
cd backend && go build ./... && go vet ./... && go test -race ./...
cd ../frontend && npm run lint && npm run build
```

Manual, against `docker compose up -d --build`:

1. `docker compose logs backend` shows `Migration applied: 006_add_user_activity_logs.sql` and `Activity log cleanup started`.
2. Signing in adds exactly one `login` row with a populated OS and browser.
3. Browsing for several minutes adds at most one `active` row per five minutes.
4. A failed sign-in adds no row.
5. The admin Users page shows a green dot and device for the signed-in admin, and "Never signed in" for an account that has not.
