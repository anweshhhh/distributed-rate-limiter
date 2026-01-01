package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupRedisTB(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	r := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})

	return r, client
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	ctx := context.Background()
	r, client := setupRedisTB(t)
	defer r.Close()

	limiter, _ := NewRedisTokenBucketLimiter(client, "test", 2, 1)
	key := "user-2"

	// Drain bucket
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Fatalf("expected deny when bucket empty")
	}

	// REAL wall-clock time advance (Go time source)
	time.Sleep(1100 * time.Millisecond)

	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Fatalf("expected request to be allowed after refill")
	}
}

func TestTokenBucket_SteadyStateRate(t *testing.T) {
	ctx := context.Background()
	r, client := setupRedisTB(t)
	defer r.Close()

	limiter, _ := NewRedisTokenBucketLimiter(client, "test", 2, 2)
	key := "user-3"

	// Consume capacity
	limiter.Allow(ctx, key)
	limiter.Allow(ctx, key)

	allowed, _ := limiter.Allow(ctx, key)
	if allowed {
		t.Fatalf("expected deny when bucket empty")
	}

	// REAL wall-clock time advance
	time.Sleep(1100 * time.Millisecond)

	allowed, _ = limiter.Allow(ctx, key)
	if !allowed {
		t.Fatalf("expected first request to be allowed after 1s refill")
	}
}
