package limiter

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisFixedWindowLimiter is a distributed fixed-window rate limiter
// backed by Redis. Behavior will be implemented incrementally.
type RedisFixedWindowLimiter struct {
	rdb        redis.UniversalClient
	namespace  string
	limit      int64
	windowSize time.Duration
	now        func() time.Time
}

// NewRedisFixedWindowLimiter creates a Redis-backed fixed window limiter.
func NewRedisFixedWindowLimiter(
	rdb redis.UniversalClient,
	namespace string,
	limit int64,
	windowSize time.Duration,
) *RedisFixedWindowLimiter {
	if rdb == nil {
		panic("redis client cannot be nil")
	}
	if limit <= 0 {
		panic("rate limit must be greater than zero")
	}
	if windowSize <= 0 {
		panic("window size must be greater than zero")
	}
	if namespace == "" {
		namespace = "fw"
	}

	return &RedisFixedWindowLimiter{
		rdb:        rdb,
		namespace:  namespace,
		limit:      limit,
		windowSize: windowSize,
		now:        time.Now,
	}
}

// Allow implements RateLimiter.
// TODO: implement Redis-backed fixed window logic.
func (r *RedisFixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	return false, nil
}
