package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock lets a test move time forward deterministically instead of
// sleeping, so these tests run in milliseconds regardless of the interval
// under test.
type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time         { return c.now }
func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// A Redis outage must be visible the moment it starts, not only after a full
// interval has passed with nothing in the log.
func TestRateLimitedLoggerAllowsTheFirstCallImmediately(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	l := newRateLimitedLogger(time.Minute, clock.Now)

	assert.True(t, l.allow(), "the first failure must log immediately")
}

// This is the regression the finding is about: without rate limiting, every
// single request during an outage would produce its own log line.
func TestRateLimitedLoggerSuppressesWithinTheInterval(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	l := newRateLimitedLogger(time.Minute, clock.Now)

	require.True(t, l.allow())
	clock.Advance(59 * time.Second)
	assert.False(t, l.allow(), "a second failure inside the window must not log again")

	clock.Advance(500 * time.Millisecond)
	assert.False(t, l.allow(), "still inside the window")
}

func TestRateLimitedLoggerAllowsAgainAfterTheInterval(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	l := newRateLimitedLogger(time.Minute, clock.Now)

	require.True(t, l.allow())
	clock.Advance(time.Minute)
	assert.True(t, l.allow(), "a failure at exactly the interval boundary may log again")
}

// TestRateLimitedLoggerDuringASustainedOutage simulates one failed request a
// second for five minutes — the shape of a real Redis outage under load —
// and checks the line count stays close to one per minute, not one per
// request.
func TestRateLimitedLoggerDuringASustainedOutage(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	l := newRateLimitedLogger(time.Minute, clock.Now)

	const outageSeconds = 5 * 60
	logged := 0
	for i := 0; i < outageSeconds; i++ {
		if l.allow() {
			logged++
		}
		clock.Advance(time.Second)
	}

	// One line at the start of the outage, then one per elapsed minute:
	// 5 minutes of continuous failures should log 5 times, never 300.
	assert.Equal(t, 5, logged)
}

func TestRateLimitedLoggerIsSafeForConcurrentUse(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	l := newRateLimitedLogger(time.Minute, clock.Now)

	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func() { done <- l.allow() }()
	}
	allowed := 0
	for i := 0; i < 20; i++ {
		if <-done {
			allowed++
		}
	}

	// Concurrent callers within the same instant must still collapse to a
	// single logged line, not one per goroutine.
	assert.Equal(t, 1, allowed)
}
