package limiter

import (
	"context"
	"testing"
	"time"
)

func TestFixedWindowLimiter(t *testing.T) {
	start := time.Now()
	current := start

	rl := NewFixedWindowLimiter(2, time.Second)
	rl.now = func() time.Time {
		return current
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		key      string
		advance  time.Duration
		allowed  bool
	}{
		{
			name:    "first request allowed",
			key:     "user1",
			allowed: true,
		},
		{
			name:    "second request allowed",
			key:     "user1",
			allowed: true,
		},
		{
			name:    "third request denied",
			key:     "user1",
			allowed: false,
		},
		{
			name:    "window resets after expiry",
			key:     "user1",
			advance: time.Second,
			allowed: true,
		},
		{
			name:    "different key has separate limit",
			key:     "user2",
			allowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.advance > 0 {
				current = current.Add(tt.advance)
			}

			allowed, err := rl.Allow(ctx, tt.key)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if allowed != tt.allowed {
				t.Fatalf("expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}
