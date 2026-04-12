package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const searchTTL = 5 * time.Minute

// Cache wraps Redis for application-level caching.
type Cache struct {
	rdb *redis.Client
}

func New(addr string) *Cache {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	return &Cache{rdb: rdb}
}

// Ping checks the Redis connection.
func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// GetSearch retrieves a cached search result. Returns (nil, nil) on cache miss.
func (c *Cache) GetSearch(ctx context.Context, key string, dest any) error {
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil // cache miss — caller checks dest for nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(val, dest)
}

// SetSearch stores a search result with the default TTL.
func (c *Cache) SetSearch(ctx context.Context, key string, val any) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, b, searchTTL).Err()
}

// SearchKey builds a canonical cache key for a route search.
func SearchKey(fromCode, toCode, date, mode string) string {
	return "search:" + fromCode + ":" + toCode + ":" + date + ":" + mode
}
