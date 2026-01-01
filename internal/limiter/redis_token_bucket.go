package limiter

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed scripts/token_bucket.lua
var tokenBucketLua string

// RedisTokenBucketLimiter enforces rate limits using a Redis-backed token bucket.
type RedisTokenBucketLimiter struct {
	rdb        *redis.Client
	namespace  string
	capacity   int
	refillRate int
	cost       int
	scale      int
	ttl        time.Duration
	script     *redis.Script
}

// NewRedisTokenBucketLimiter constructs a Redis-backed token bucket limiter.
//
// capacity   : maximum burst size
// refillRate : tokens per second
func NewRedisTokenBucketLimiter(
	rdb *redis.Client,
	namespace string,
	capacity int,
	refillRate int,
) (*RedisTokenBucketLimiter, error) {

	if rdb == nil {
		return nil, errors.New("redis client is nil")
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("invalid capacity: %d", capacity)
	}
	if refillRate <= 0 {
		return nil, fmt.Errorf("invalid refill rate: %d", refillRate)
	}

	// TTL derived from refill characteristics:
	// ttl ≈ 2 × (capacity / refillRate)
	ttlSeconds := int((2 * capacity) / refillRate)
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	return &RedisTokenBucketLimiter{
		rdb:        rdb,
		namespace:  namespace,
		capacity:   capacity,
		refillRate: refillRate,
		cost:       1,
		scale:      1_000_000,
		ttl:        time.Duration(ttlSeconds) * time.Second,
		script:     redis.NewScript(tokenBucketLua),
	}, nil
}

// Allow implements the RateLimiter interface.
func (l *RedisTokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, errors.New("empty key")
	}

	redisKey := l.redisKey(key)

	res, err := l.script.Run(
		ctx,
		l.rdb,
		[]string{redisKey},
		l.capacity,
		l.refillRate,
		l.cost,
		l.scale,
		l.ttl.Milliseconds(),
	).Result()

	if err != nil {
		// Fail closed on Redis/Lua errors
		return false, err
	}

	values, ok := res.([]interface{})
	if !ok || len(values) < 1 {
		return false, errors.New("unexpected lua return value")
	}

	allowed, ok := values[0].(int64)
	if !ok {
		return false, errors.New("invalid lua return type")
	}

	return allowed == 1, nil
}

// redisKey builds the Redis key for a given identity.
func (l *RedisTokenBucketLimiter) redisKey(key string) string {
	return fmt.Sprintf(
		"rl:%s:%s:tb",
		l.namespace,
		hashKey(key),
	)
}
