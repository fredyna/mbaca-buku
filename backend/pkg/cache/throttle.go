package cache

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisOutageLogInterval caps how often a Redis failure is logged.
// ActivityService.Record swallows a "not allowed" answer silently (see the
// spec's "Kegagalan Redis ... dicatat ke log server dan diabaikan"), so an
// unthrottled line per request would flood the log during exactly the
// incident an operator most needs to read — one line a minute is enough to
// notice the outage without drowning everything else out.
const redisOutageLogInterval = time.Minute

// rateLimitedLogger decides whether "now" is allowed to log, at most once per
// interval. It holds no reference to Redis or anything else being logged
// about, so its decision logic can be tested on its own.
type rateLimitedLogger struct {
	mu       sync.Mutex
	interval time.Duration
	now      func() time.Time
	last     time.Time // zero value: nothing logged yet
}

func newRateLimitedLogger(interval time.Duration, now func() time.Time) *rateLimitedLogger {
	return &rateLimitedLogger{interval: interval, now: now}
}

// allow reports whether a line may be logged right now, claiming the window
// when it says yes. The very first call always succeeds — an outage's first
// failure is exactly the line an operator needs to see immediately, not
// after waiting out a full interval.
func (l *rateLimitedLogger) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if !l.last.IsZero() && now.Sub(l.last) < l.interval {
		return false
	}
	l.last = now
	return true
}

// RedisThrottle answers "may this key act now?" in a single SET NX EX round
// trip. Activity logging uses it so that an idle or busy user costs one Redis
// call rather than a database read.
type RedisThrottle struct {
	rdb       *redis.Client
	errLogger *rateLimitedLogger
}

func NewRedisThrottle(rdb *redis.Client) *RedisThrottle {
	return &RedisThrottle{
		rdb:       rdb,
		errLogger: newRateLimitedLogger(redisOutageLogInterval, time.Now),
	}
}

// Allow reports whether the caller may act, claiming the window when it says
// yes. A Redis failure answers no: losing an activity row is harmless, while
// treating an outage as "allowed" would write a row on every single request.
//
// The failure is also logged, rate-limited via errLogger so a sustained
// outage produces roughly one line a minute instead of one per request —
// otherwise the log flood would drown out the very incident it should be
// reporting. The activity log's own silence during an outage (ActivityService
// just returns) would otherwise leave nothing in the logs explaining why
// last_active_at stopped moving.
func (t *RedisThrottle) Allow(ctx context.Context, key string, window time.Duration) bool {
	claimed, err := t.rdb.SetNX(ctx, key, 1, window).Result()
	if err != nil {
		if t.errLogger.allow() {
			log.Printf("activity throttle: redis unavailable, activity recording is degraded: %v", err)
		}
		return false
	}
	return claimed
}
