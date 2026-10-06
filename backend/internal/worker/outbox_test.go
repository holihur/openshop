package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/port"
)

type stubOutbox struct {
	mu        sync.Mutex
	claimed   []port.OutboxMessage
	published map[string]bool
	failed    map[string]string
	reclaimed int64
}

func newStubOutbox(claimed ...port.OutboxMessage) *stubOutbox {
	return &stubOutbox{claimed: claimed, published: map[string]bool{}, failed: map[string]string{}}
}

func (o *stubOutbox) Enqueue(context.Context, port.Event) error { return nil }
func (o *stubOutbox) Claim(context.Context, int, time.Duration) ([]port.OutboxMessage, error) {
	return o.claimed, nil
}
func (o *stubOutbox) MarkPublished(_ context.Context, id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published[id] = true
	return nil
}
func (o *stubOutbox) MarkFailed(_ context.Context, msg port.OutboxMessage, reason string, _ time.Time) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.failed[msg.ID] = reason
	return nil
}
func (o *stubOutbox) Reclaim(context.Context, time.Time) (int64, error) { return o.reclaimed, nil }

type stubBus struct {
	mu        sync.Mutex
	fail      map[string]bool
	published []string
}

func (b *stubBus) Publish(_ context.Context, evt port.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fail[evt.ID] {
		return errors.New("broker unavailable")
	}
	b.published = append(b.published, evt.ID)
	return nil
}
func (b *stubBus) Subscribe(string, string, string, port.EventHandler) error { return nil }
func (b *stubBus) Close() error                                              { return nil }

type stubLogger struct{}

func (stubLogger) Debug(string, ...any)    {}
func (stubLogger) Info(string, ...any)     {}
func (stubLogger) Warn(string, ...any)     {}
func (stubLogger) Error(string, ...any)    {}
func (stubLogger) With(...any) port.Logger { return stubLogger{} }

func TestOutboxRelayPublishesAndRetries(t *testing.T) {
	outbox := newStubOutbox(
		port.OutboxMessage{ID: "ok", Subject: "order.created", Payload: []byte("{}"), Attempts: 1},
		port.OutboxMessage{ID: "boom", Subject: "order.paid", Payload: []byte("{}"), Attempts: 1},
	)
	bus := &stubBus{fail: map[string]bool{"boom": true}}

	relay := NewOutboxRelay(outbox, bus, port.SystemClock{}, stubLogger{}, port.NopMetrics{}, port.NoopTracer{}, time.Second, 10)
	relay.relayOnce(context.Background())

	if !outbox.published["ok"] {
		t.Fatal("successful message was not marked published")
	}
	if _, ok := outbox.failed["boom"]; !ok {
		t.Fatal("failed message was not rescheduled")
	}
	if len(bus.published) != 1 || bus.published[0] != "ok" {
		t.Fatalf("published = %v, want [ok]", bus.published)
	}
}

func TestOutboxRelayBackoffGrowsAndCaps(t *testing.T) {
	relay := NewOutboxRelay(newStubOutbox(), &stubBus{}, port.SystemClock{}, stubLogger{}, port.NopMetrics{}, port.NoopTracer{}, time.Second, 10)

	if got := relay.backoff(1); got != time.Second {
		t.Fatalf("backoff(1) = %v, want 1s", got)
	}
	if got := relay.backoff(3); got != 4*time.Second {
		t.Fatalf("backoff(3) = %v, want 4s", got)
	}
	if got := relay.backoff(20); got != relay.maxDelay {
		t.Fatalf("backoff(20) = %v, want cap %v", got, relay.maxDelay)
	}
}
