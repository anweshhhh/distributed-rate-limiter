package limiter

import (
    "context"
    "sync"
    "time"
)


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


func NewFixedWindowLimiter(limit int64, windowSize time.Duration) *FixedWindowLimiter {
	if limit <= 0 {
		panic("rate limit must be greater than zero")
	}

	if windowSize <= 0 {
		panic("window size must be greater than zero")
	}

	return &FixedWindowLimiter{
		limit:      limit,
		windowSize: windowSize,
		states:     make(map[string]*fixedWindowState),
		now:        time.Now,
	}
}


func (f *FixedWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.now().UnixNano()

	state, exists := f.states[key]
	if !exists {
		f.states[key] = &fixedWindowState{
			windowStart: now,
			count:       1,
		}
		return true, nil
	}

	windowEnd := state.windowStart + f.windowSize.Nanoseconds()
	if now >= windowEnd {
		state.windowStart = now
		state.count = 1
		return true, nil
	}

	if state.count < f.limit {
		state.count++
		return true, nil
	}

	return false, nil
}


