package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSlidingWindowVsFixedWindowBoundaryBurst(t *testing.T) {
	ctx := context.Background()

	r := miniredis.RunT(t)
	defer r.Close()

	client := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})

	fixed := NewRedisFixedWindowLimiter(
		client,
		"test",
		4,
		time.Second,
	)

	// Override clock so fixed window follows Redis time
	now := time.Unix(0, 0)
	fixed.now = func() time.Time { return now }

	sliding, err := NewRedisSlidingWindowCounter(
		client,
		"test",
		4,
		1000,
	)
	if err != nil {
		t.Fatalf("failed to create sliding limiter: %v", err)
	}

	key := "user123"

	// Late in first window
	r.FastForward(900 * time.Millisecond)
	now = now.Add(900 * time.Millisecond)

	fixedAllowed := 0
	slidingAllowed := 0

	for i := 0; i < 4; i++ {
		ok, _ := fixed.Allow(ctx, key)
		if ok {
			fixedAllowed++
		}

		ok, _ = sliding.Allow(ctx, key)
		if ok {
			slidingAllowed++
		}
	}

	// Cross window boundary
	r.FastForward(200 * time.Millisecond)
	now = now.Add(200 * time.Millisecond)

	for i := 0; i < 4; i++ {
		ok, _ := fixed.Allow(ctx, key)
		if ok {
			fixedAllowed++
		}

		ok, _ = sliding.Allow(ctx, key)
		if ok {
			slidingAllowed++
		}
	}

	if fixedAllowed != 8 {
		t.Fatalf("expected fixed window to allow full burst, got %d", fixedAllowed)
	}

	if slidingAllowed >= fixedAllowed {
		t.Fatalf(
			"expected sliding window to smooth burst: fixed=%d sliding=%d",
			fixedAllowed,
			slidingAllowed,
		)
	}
}
