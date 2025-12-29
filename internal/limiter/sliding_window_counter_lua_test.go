package limiter

import (
	"os"
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSlidingWindowCounterLua(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	defer r.Close()

	client := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})

	script, err := os.ReadFile("internal/limiter/scripts/sliding_window_counter.lua")
	if err != nil {
		t.Fatalf("failed to read lua script: %v", err)
	}

	lua := redis.NewScript(string(script))

	const (
		limit    = 3
		windowMs = 1000
		ttlMs    = 3000
	)

	key := "user123"
	currKey := "rl:test:swc:curr"
	prevKey := "rl:test:swc:prev"

	// --- steady state allow ---
	for i := 0; i < limit; i++ {
		res, err := lua.Run(ctx, client, []string{currKey, prevKey},
			limit, windowMs, ttlMs,
		).Int()
		if err != nil {
			t.Fatalf("lua run failed: %v", err)
		}
		if res != 1 {
			t.Fatalf("expected allow on iteration %d", i)
		}
	}

	// --- deny when limit exceeded ---
	res, _ := lua.Run(ctx, client, []string{currKey, prevKey},
		limit, windowMs, ttlMs,
	).Int()
	if res != 0 {
		t.Fatalf("expected deny after limit reached")
	}

	// --- advance time into next window ---
	r.FastForward(time.Second + 10*time.Millisecond)

	res, _ = lua.Run(ctx, client, []string{currKey, prevKey},
		limit, windowMs, ttlMs,
	).Int()
	if res != 1 {
		t.Fatalf("expected allow after window slide")
	}
}

func TestSlidingWindowCounterBoundarySmoothing(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	defer r.Close()

	client := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})

	script, _ := os.ReadFile("internal/limiter/scripts/sliding_window_counter.lua")
	lua := redis.NewScript(string(script))

	const (
		limit    = 4
		windowMs = 1000
		ttlMs    = 3000
	)

	currKey := "rl:test:swc:curr"
	prevKey := "rl:test:swc:prev"

	// Send requests late in window
	r.FastForward(900 * time.Millisecond)
	for i := 0; i < 3; i++ {
		lua.Run(ctx, client, []string{currKey, prevKey},
			limit, windowMs, ttlMs,
		)
	}

	// Cross window boundary
	r.FastForward(200 * time.Millisecond)

	allowed := 0
	for i := 0; i < 4; i++ {
		res, _ := lua.Run(ctx, client, []string{currKey, prevKey},
			limit, windowMs, ttlMs,
		).Int()
		if res == 1 {
			allowed++
		}
	}

	if allowed >= limit {
		t.Fatalf("expected boundary smoothing to reduce burstiness")
	}
}
