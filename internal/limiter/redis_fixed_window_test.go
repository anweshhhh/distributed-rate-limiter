package limiter

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

func TestRedisFixedWindowLimiter_AllowDenyAndReset(t *testing.T) {
	// Start in-memory Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	limit := int64(3)
	window := 10 * time.Second

	// Controlled clock
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	limiter := NewRedisFixedWindowLimiter(
		rdb,
		"fw",
		limit,
		window,
	)

	// Inject fake clock
	limiter.now = func() time.Time {
		return now
	}

	ctx := context.Background()

	// First 3 requests allowed
	for i := 0; i < int(limit); i++ {
		allowed, err := limiter.Allow(ctx, "userA")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	// 4th request denied
	allowed, err := limiter.Allow(ctx, "userA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatalf("expected request to be denied after limit reached")
	}

	// Move time into next window
	now = now.Add(window)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, "userA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected request to be allowed after window reset")
	}
}
