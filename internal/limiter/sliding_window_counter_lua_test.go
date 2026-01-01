package limiter

import (
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

	lua := redis.NewScript(slidingWindowCounterLua)

	const (
		limit    = 3
		windowMs = 1000
		ttlMs    = 3000
	)

	baseKey := "rl:test:user123:swc"

	for i := 0; i < limit; i++ {
		res, _ := lua.Run(ctx, client, []string{baseKey}, limit, windowMs, ttlMs).Int()
		if res != 1 {
			t.Fatalf("expected allow")
		}
	}

	res, _ := lua.Run(ctx, client, []string{baseKey}, limit, windowMs, ttlMs).Int()
	if res != 0 {
		t.Fatalf("expected deny")
	}

	// REAL TIME ADVANCE
	time.Sleep(1200 * time.Millisecond)

	res, _ = lua.Run(ctx, client, []string{baseKey}, limit, windowMs, ttlMs).Int()
	if res != 1 {
		t.Fatalf("expected allow after window slide")
	}
}
