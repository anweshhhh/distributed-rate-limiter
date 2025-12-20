package limiter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisFixedWindowLimiter is a distributed fixed-window rate limiter
// backed by Redis.
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
// Redis logic will be added in later commits.
func (r *RedisFixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	_ = r.currentWindowStart()
	_ = r.redisKey(key)
	return false, nil
}

/*
NEW BELOW
*/

// currentWindowStart returns the unix timestamp (seconds)
// of the current fixed window start.
func (r *RedisFixedWindowLimiter) currentWindowStart() int64 {
	windowSec := int64(r.windowSize.Seconds())
	nowUnix := r.now().Unix()
	return (nowUnix / windowSec) * windowSec
}

// redisKey returns the Redis key for the given logical key
// in the current window.
func (r *RedisFixedWindowLimiter) redisKey(key string) string {
	start := r.currentWindowStart()

	h := sha1.Sum([]byte(key))
	keyHash := hex.EncodeToString(h[:])

	return fmt.Sprintf("rl:%s:%s:%d", r.namespace, keyHash, start)
}
