package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ResultCache caches serialised query results in Redis.
type ResultCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *ResultCache {
	return &ResultCache{rdb: rdb, ttl: ttl}
}

func (c *ResultCache) Get(ctx context.Context, key string) (string, bool) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *ResultCache) Set(ctx context.Context, key, value string) {
	_ = c.rdb.Set(ctx, key, value, c.ttl).Err()
}

func CountKey(tenantID, event string, from, to int64) string {
	return fmt.Sprintf("query:count:%s:%s:%d:%d", tenantID, event, from, to)
}

func TimeSeriesKey(tenantID, event, interval string, from, to int64) string {
	return fmt.Sprintf("query:ts:%s:%s:%s:%d:%d", tenantID, event, interval, from, to)
}
