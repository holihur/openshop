package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/holihur/openshop/internal/port"
)

// Cache implements port.Cache on top of Redis.
type Cache struct {
	rdb *goredis.Client
}

func NewCache(c *Client) *Cache { return &Cache{rdb: c.Raw()} }

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", port.ErrCacheMiss
	}
	return v, err
}

func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl <= 0 {
		return c.rdb.Set(ctx, key, value, 0).Err()
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest any) error {
	raw, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Set(ctx, key, string(raw), ttl)
}

func (c *Cache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return c.rdb.IncrBy(ctx, key, delta).Result()
}

func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Expire(ctx, key, ttl).Err()
}

var _ port.Cache = (*Cache)(nil)
