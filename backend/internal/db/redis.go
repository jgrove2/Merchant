package db

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedis initializes a new Redis client
// addr should be in the format "host:port", e.g. "localhost:6379"
func NewRedis(addr string) (*Redis, error) {
	if addr == "" {
		addr = "localhost"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()

	// Ping to verify connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Redis{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Add sets a key-value pair in Redis.
// It uses a default expiration of 0 (no expiration) if not specified differently in future extensions.
func (r *Redis) Add(key string, value interface{}) error {
	// For basic "Add", we'll just set the value with no expiration
	return r.client.Set(r.ctx, key, value, 0).Err()
}

// AddWithTTL sets a key-value pair in Redis with a specific time-to-live.
func (r *Redis) AddWithTTL(key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(r.ctx, key, value, ttl).Err()
}

// Remove deletes a key from Redis.
func (r *Redis) Remove(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

// Get retrieves a string value from Redis.
func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}
