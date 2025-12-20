package limiter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// FailureMode defines behavior when Redis is unavailable.
type FailureMode int

const (
	FailOpen FailureMode = iota
	FailClosed
)

// RedisFixedWindowLimiter is a distributed fixed-window rate limiter
// backed by Redis.
type RedisFixedWindowLimiter struct {
	rdb        redis.UniversalClient
	namespace  string
	limit      int64
	windowSize time.Duration
	now        func() time.Time

	failureMode FailureMode
	script      *redis.Script
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

	lua := `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], tonumber(ARGV[2]))
end

local limit = tonumber(ARGV[1])
if current <= limit then
  return 1
end
return 0
`

	return &RedisFixedWindowLimiter{
		rdb:         rdb,
		namespace:   namespace,
		limit:       limit,
		windowSize:  windowSize,
		now:         time.Now,
		failureMode: FailClosed,
		script:      redis.NewScript(lua),
	}
}

// Allow implements RateLimiter using Redis-backed fixed window logic.
func (r *RedisFixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := r.redisKey(key)

	windowSec := int64(r.windowSize.Seconds())
	ttlSeconds := windowSec + 1 // buffer to avoid boundary edge cases

	result, err := r.script.Run(
		ctx,
		r.rdb,
		[]string{redisKey},
		r.limit,
		ttlSeconds,
	).Int64()

	if err != nil {
		if r.failureMode == FailOpen {
			return true, nil
		}
		return false, err
	}

	return result == 1, nil
}

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
