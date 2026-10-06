// Package redis contains shared-state adapters backed by Redis. Every adapter
// is used only through a port interface, so the application can run many
// replicas that all observe the same cache, locks and carts.
package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/holihur/openshop/internal/config"
)

// Client is a thin lifecycle wrapper around the official Redis client. It is
// created once per process and shared by all adapters.
type Client struct {
	rdb *goredis.Client
}

func NewClient(cfg config.RedisConfig) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Raw() *goredis.Client { return c.rdb }

// Ping reports connectivity for readiness probes.
func (c *Client) Ping(ctx context.Context) error { return c.rdb.Ping(ctx).Err() }

func (c *Client) Close() error { return c.rdb.Close() }
