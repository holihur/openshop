package port

import (
	"context"
	"time"
)

// Cache abstracts a shared, out-of-process cache (Redis in production). Every
// instance must observe the same values, so an in-memory map is not a valid
// implementation when running horizontally scaled.
type Cache interface {
	Get(ctx context.Context, key string) (string, error) // ErrCacheMiss when absent
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Delete removes keys. Implementations should treat missing keys as success.
	Delete(ctx context.Context, keys ...string) error
	// GetJSON/SetJSON are convenience helpers for structured values.
	GetJSON(ctx context.Context, key string, dest any) error
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error
	// Incr is an atomic counter used for rate limiting and ID sequences.
	Incr(ctx context.Context, key string, delta int64) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

// Locker provides distributed mutual exclusion. Because checkout runs on many
// instances, correctness cannot rely on a process-local mutex.
type Locker interface {
	// Acquire tries to take the lock, blocking up to wait time. It returns a
	// Lock handle on success or ErrLockUnavailable.
	Acquire(ctx context.Context, key string, ttl, wait time.Duration) (Lock, error)
}

// Lock is a held distributed lock. Release must be idempotent and must only
// release the lock when the caller still owns it (token check).
type Lock interface {
	Release(ctx context.Context) error
}

// ErrCacheMiss is returned by Cache implementations when a key is absent.
var ErrCacheMiss = errCacheMiss{}

type errCacheMiss struct{}

func (errCacheMiss) Error() string { return "cache: key not found" }
