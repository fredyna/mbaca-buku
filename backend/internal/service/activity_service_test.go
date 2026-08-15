package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fredy/mbaca-buku/internal/model"
)

// fakeActivityStore records what was written rather than which methods were
// called, so tests can assert on the parsed OS/browser landing in the row.
type fakeActivityStore struct {
	mu        sync.Mutex
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
	f.mu.Lock()
	defer f.mu.Unlock()
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
	for {
		store.mu.Lock()
		calls := store.deleteCalls
		store.mu.Unlock()
		if calls > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Equal(t, 1, store.deleteCalls, "cleanup must run at startup, not only on the next tick")
	assert.Equal(t, 90*24*time.Hour, store.deletedAge)
}
