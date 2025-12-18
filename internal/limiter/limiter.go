package limiter

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type fixedWindowState struct {
    windowStart int64
    count       int64
}
