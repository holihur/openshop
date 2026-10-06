package redis

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// Locker implements port.Locker using Redis SET NX with an expiring token.
// The token makes release owner-safe: a lock can only be freed by the process
// that acquired it, even if the TTL already elapsed and another instance took
// over.
type Locker struct {
	rdb *goredis.Client
}

func NewLocker(c *Client) *Locker { return &Locker{rdb: c.Raw()} }

// releaseScript deletes the key only when its value matches the caller token.
var releaseScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

type lock struct {
	rdb   *goredis.Client
	key   string
	token string
}

func (l *Locker) Acquire(ctx context.Context, key string, ttl, wait time.Duration) (port.Lock, error) {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	token := uuid.NewString()
	deadline := time.Now().Add(wait)

	for {
		ok, err := l.rdb.SetNX(ctx, key, token, ttl).Result()
		if err != nil {
			return nil, err
		}
		if ok {
			return &lock{rdb: l.rdb, key: key, token: token}, nil
		}
		if time.Now().After(deadline) {
			return nil, domain.ErrLockUnavailable
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (l *lock) Release(ctx context.Context) error {
	_, err := releaseScript.Run(ctx, l.rdb, []string{l.key}, l.token).Result()
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	return err
}

var _ port.Locker = (*Locker)(nil)
