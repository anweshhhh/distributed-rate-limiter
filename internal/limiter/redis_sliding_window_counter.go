package limiter

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

type RedisSlidingWindowCounter struct {
	client   *redis.Client
	limit    int
	windowMs int
	ttlMs    int
	lua      *redis.Script
	namespace string
}

func NewRedisSlidingWindowCounter(
	client *redis.Client,
	namespace string,
	limit int,
	windowMs int,
) (*RedisSlidingWindowCounter, error) {

	script, err := os.ReadFile("internal/limiter/scripts/sliding_window_counter.lua")
	if err != nil {
		return nil, err
	}

	return &RedisSlidingWindowCounter{
		client:    client,
		limit:     limit,
		windowMs:  windowMs,
		ttlMs:     (2 * windowMs) + 1000,
		lua:       redis.NewScript(string(script)),
		namespace: namespace,
	}, nil
}

func (r *RedisSlidingWindowCounter) Allow(ctx context.Context, key string) (bool, error) {
	hash := sha1.Sum([]byte(key))
	hashedKey := hex.EncodeToString(hash[:])

	base := fmt.Sprintf("rl:%s:%s:swc", r.namespace, hashedKey)

	currKey := base + ":curr"
	prevKey := base + ":prev"

	res, err := r.lua.Run(
		ctx,
		r.client,
		[]string{currKey, prevKey},
		r.limit,
		r.windowMs,
		r.ttlMs,
	).Int()

	if err != nil {
		return false, err
	}

	return res == 1, nil
}
