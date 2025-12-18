package limiter

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type fixedWindowState struct {
    windowStart int64
    count       int64
}

type FixedWindowLimiter struct {
    mu         sync.Mutex
    states     map[string]*fixedWindowState
    limit      int64
    windowSize time.Duration
    now        func() time.Time // optional but recommended
}
