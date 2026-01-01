package limiter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/redis/go-redis/v9"

	_ "embed"
)

//go:embed scripts/sliding_window_counter.lua
var slidingWindowCounterLua string

type RedisSlidingWindowCounter struct {
	client    *redis.Client
	limit     int
	windowMs  int
	ttlMs     int
	lua       *redis.Script
	namespace string
}

func NewRedisSlidingWindowCounter(
	client *redis.Client,
	namespace string,
	limit int,
	windowMs int,
) (*RedisSlidingWindowCounter, error) {

	return &RedisSlidingWindowCounter{
		client:    client,
		limit:     limit,
		windowMs:  windowMs,
		ttlMs:     (2 * windowMs) + 1000,
		lua:       redis.NewScript(slidingWindowCounterLua),
		namespace: namespace,
	}, nil
}

func (r *RedisSlidingWindowCounter) Allow(ctx context.Context, key string) (bool, error) {
	hash := sha1.Sum([]byte(key))
	hashedKey := hex.EncodeToString(hash[:])

	baseKey := fmt.Sprintf("rl:%s:%s:swc", r.namespace, hashedKey)

	res, err := r.lua.Run(
		ctx,
		r.client,
		[]string{baseKey},
		r.limit,
		r.windowMs,
		r.ttlMs,
	).Int()

	if err != nil {
		return false, err
	}

	return res == 1, nil
}
