package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
)

// openTestClient connects to the Redis named by REDIS_ADDR (or localhost:6379)
// and skips the test when it is unreachable, so a plain `go test ./...` still
// runs the pure unit tests without any infrastructure.
func openTestClient(t *testing.T) *Client {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client, err := NewClient(config.RedisConfig{Addr: addr, PoolSize: 5})
	if err != nil {
		t.Skipf("redis not reachable at %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func uniqueKey(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("test:%s:%d", prefix, time.Now().UnixNano())
}

// The limiter is a sliding window over shared state: two instances must observe
// the same budget, which is what makes the limit hold across replicas.
func TestRateLimiterEnforcesSharedLimit(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	key := uniqueKey(t, "ratelimit")
	t.Cleanup(func() { _ = client.Raw().Del(context.Background(), key) })

	replicaA := NewRateLimiter(client)
	replicaB := NewRateLimiter(client)
	const limit = 3

	for i := 0; i < limit; i++ {
		allowed, _, err := replicaA.Allow(ctx, key, limit, time.Minute)
		if err != nil {
			t.Fatalf("allow %d: %v", i, err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed (limit %d)", i, limit)
		}
	}

	// The budget is exhausted, and the other replica sees the same counter.
	allowed, retry, err := replicaB.Allow(ctx, key, limit, time.Minute)
	if err != nil {
		t.Fatalf("allow over limit: %v", err)
	}
	if allowed {
		t.Fatal("the request over the limit must be refused")
	}
	if retry <= 0 {
		t.Fatalf("a refusal must suggest a positive retry delay, got %v", retry)
	}
}

func TestRateLimiterWindowExpires(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	key := uniqueKey(t, "ratelimit-window")
	t.Cleanup(func() { _ = client.Raw().Del(context.Background(), key) })

	limiter := NewRateLimiter(client)
	window := 300 * time.Millisecond

	if allowed, _, _ := limiter.Allow(ctx, key, 1, window); !allowed {
		t.Fatal("the first request should be allowed")
	}
	if allowed, _, _ := limiter.Allow(ctx, key, 1, window); allowed {
		t.Fatal("the second request inside the window must be refused")
	}

	time.Sleep(window + 100*time.Millisecond)
	if allowed, _, err := limiter.Allow(ctx, key, 1, window); err != nil || !allowed {
		t.Fatalf("after the window the request should be allowed again (err=%v)", err)
	}
}

func TestRateLimiterDisabledAllowsEverything(t *testing.T) {
	client := openTestClient(t)
	limiter := NewRateLimiter(client)
	for i := 0; i < 100; i++ {
		allowed, _, err := limiter.Allow(context.Background(), uniqueKey(t, "ratelimit-off"), 0, time.Second)
		if err != nil || !allowed {
			t.Fatalf("a zero limit must allow every request (err=%v allowed=%v)", err, allowed)
		}
	}
}

// A lock is mutually exclusive: the second holder cannot take it while the first
// is live, and can once it is released.
func TestLockerIsMutuallyExclusive(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	locker := NewLocker(client)
	key := uniqueKey(t, "lock")

	first, err := locker.Acquire(ctx, key, 10*time.Second, 0)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := locker.Acquire(ctx, key, 10*time.Second, 0); !errors.Is(err, domain.ErrLockUnavailable) {
		t.Fatalf("a second holder must be refused, got %v", err)
	}
	if err := first.Release(ctx); err != nil {
		t.Fatalf("release: %v", err)
	}
	second, err := locker.Acquire(ctx, key, 10*time.Second, 0)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	_ = second.Release(ctx)
}

// The release token makes release owner-safe: a holder whose lease already
// expired must not delete the lock a new holder took over.
func TestLockerReleaseIsOwnerSafe(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	locker := NewLocker(client)
	key := uniqueKey(t, "lock-owner")

	stale, err := locker.Acquire(ctx, key, 200*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	time.Sleep(350 * time.Millisecond) // let the lease lapse

	fresh, err := locker.Acquire(ctx, key, 5*time.Second, 0)
	if err != nil {
		t.Fatalf("acquire after expiry: %v", err)
	}

	// The stale holder releases late; it must not free someone else's lock.
	if err := stale.Release(ctx); err != nil {
		t.Fatalf("stale release: %v", err)
	}
	if _, err := locker.Acquire(ctx, key, time.Second, 0); !errors.Is(err, domain.ErrLockUnavailable) {
		t.Fatalf("the new holder must still own the lock, got %v", err)
	}
	_ = fresh.Release(ctx)
}

// A caller that is willing to wait gets the lock once it is freed.
func TestLockerWaitsForRelease(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	locker := NewLocker(client)
	key := uniqueKey(t, "lock-wait")

	holder, err := locker.Acquire(ctx, key, 5*time.Second, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = holder.Release(context.Background())
	}()

	waiter, err := locker.Acquire(ctx, key, 5*time.Second, 2*time.Second)
	if err != nil {
		t.Fatalf("a waiting caller should get the lock: %v", err)
	}
	_ = waiter.Release(ctx)
}

func TestLockerRespectsContextCancellation(t *testing.T) {
	client := openTestClient(t)
	locker := NewLocker(client)
	key := uniqueKey(t, "lock-cancel")

	holder, err := locker.Acquire(context.Background(), key, 5*time.Second, 0)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer func() { _ = holder.Release(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if _, err := locker.Acquire(ctx, key, 5*time.Second, 5*time.Second); err == nil {
		t.Fatal("a cancelled wait must not return a lock")
	}
}

func TestCacheSetGetDelete(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	cache := NewCache(client)
	key := uniqueKey(t, "cache")

	if err := cache.Set(ctx, key, "value", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := cache.Get(ctx, key)
	if err != nil || got != "value" {
		t.Fatalf("get = %q err=%v, want value", got, err)
	}
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := cache.Get(ctx, key); err == nil {
		t.Fatal("a deleted key must not be found")
	}
}

func TestCacheJSONRoundTrip(t *testing.T) {
	client := openTestClient(t)
	ctx := context.Background()
	cache := NewCache(client)
	key := uniqueKey(t, "cache-json")

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	if err := cache.SetJSON(ctx, key, payload{Name: "a", Count: 2}, time.Minute); err != nil {
		t.Fatalf("set json: %v", err)
	}
	var out payload
	if err := cache.GetJSON(ctx, key, &out); err != nil {
		t.Fatalf("get json: %v", err)
	}
	if out.Name != "a" || out.Count != 2 {
		t.Fatalf("round trip = %+v", out)
	}
}
