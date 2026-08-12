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
