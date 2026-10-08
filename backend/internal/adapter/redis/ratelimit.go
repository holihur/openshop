package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/holihur/openshop/internal/port"
)

// RateLimiter implements port.RateLimiter as a Redis sorted-set sliding window.
// Each request adds a member scored by its timestamp; the count of members in
// the trailing window is the current rate. Because state lives in Redis, the
// limit is enforced across every replica.
type RateLimiter struct {
	rdb *goredis.Client
}

func NewRateLimiter(c *Client) *RateLimiter { return &RateLimiter{rdb: c.Raw()} }

// slidingWindowScript prunes expired members, then admits or rejects atomically
// so concurrent requests from different replicas cannot both slip past the
// limit.
var slidingWindowScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  local retry = 0
  if oldest[2] then retry = (tonumber(oldest[2]) + window) - now end
  if retry < 0 then retry = 0 end
  return {0, retry}
end
redis.call('ZADD', key, now, member)
redis.call('PEXPIRE', key, window)
return {1, 0}
`)

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		return true, 0, nil
	}
	nowMs := time.Now().UnixMilli()
	// The member only has to be unique within the sorted set; a monotonic
	// nanosecond stamp is enough and avoids a (non-cryptographic) RNG.
	member := fmt.Sprintf("%d-%d", nowMs, time.Now().UnixNano())

	res, err := slidingWindowScript.Run(ctx, r.rdb, []string{key},
		nowMs, window.Milliseconds(), limit, member).Result()
	if err != nil {
		return false, 0, err
	}
	vals, ok := res.([]interface{})
	if !ok || len(vals) < 2 {
		return false, 0, fmt.Errorf("redis: unexpected rate limiter response %v", res)
	}
	allowed := asInt(vals[0]) == 1
	return allowed, time.Duration(asInt(vals[1])) * time.Millisecond, nil
}

func asInt(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case string:
		var out int64
		_, _ = fmt.Sscan(n, &out)
		return out
	default:
		return 0
	}
}

var _ port.RateLimiter = (*RateLimiter)(nil)
