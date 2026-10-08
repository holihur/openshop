package port

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// Event is a transport-agnostic message published to the event bus.
type Event struct {
	// Subject is the routing key, e.g. "order.created".
	Subject string
	// ID enables consumer-side idempotency.
	ID string
	// Payload is the JSON-encoded body.
	Payload []byte
	// TraceParent is a W3C traceparent carried across the async boundary so a
	// checkout and the worker reacting to it share one trace.
	TraceParent string
}

// EventBus abstracts a message broker (NATS JetStream in production). It
// decouples write paths from side effects so the API stays fast and instances
// can scale independently.
type EventBus interface {
	Publish(ctx context.Context, evt Event) error
	// Subscribe registers a durable, queue-group consumer so exactly one member
	// of the group processes each message across all instances.
	Subscribe(subject, queue, durable string, handler EventHandler) error
	Close() error
}

// EventHandler processes a single event. Returning an error triggers a retry
// according to the broker policy.
type EventHandler func(ctx context.Context, evt Event) error

// OutboxMessage is a persisted event awaiting publication.
type OutboxMessage struct {
	ID          string
	Subject     string
	Payload     []byte
	Attempts    int
	TraceParent string
	// Status is pending, processing, published or failed (dead-lettered).
	Status      string
	LastError   string
	CreatedAt   time.Time
	AvailableAt time.Time
}

// OutboxFilter selects outbox rows for the operations console.
type OutboxFilter struct {
	Status   string
	Subject  string
	Page     int
	PageSize int
}

// OutboxStats summarises the queue. Failed is the dead-letter count: events the
// relay gave up on after the maximum number of attempts.
type OutboxStats struct {
	Pending       int64
	Processing    int64
	Published     int64
	Failed        int64
	OldestPending *time.Time
}

// OutboxInspector exposes the queue to operators: what is waiting, what has
// dead-lettered, and a way to put a dead letter back. It is deliberately
// separate from Outbox so the relay's interface stays minimal.
type OutboxInspector interface {
	List(ctx context.Context, f OutboxFilter) (domain.Page[OutboxMessage], error)
	Stats(ctx context.Context) (OutboxStats, error)
	// Replay returns a dead-lettered event to the pending queue.
	Replay(ctx context.Context, id string) error
}

// Outbox implements the transactional-outbox pattern: events are written in the
// same database transaction as the state change that produced them, then a
// relay publishes them to the broker. This guarantees at-least-once delivery
// without distributed transactions and without losing events on a crash.
type Outbox interface {
	// Enqueue persists an event. It must join the ambient transaction when the
	// context carries one, so the event is committed atomically with the state.
	Enqueue(ctx context.Context, evt Event) error
	// Claim atomically leases up to limit pending messages for the caller. It is
	// safe to run on many replicas concurrently (SKIP LOCKED semantics).
	Claim(ctx context.Context, limit int, lease time.Duration) ([]OutboxMessage, error)
	// MarkPublished removes a message from the pending set.
	MarkPublished(ctx context.Context, id string) error
	// MarkFailed schedules a retry with backoff, or parks the message after the
	// maximum number of attempts.
	MarkFailed(ctx context.Context, msg OutboxMessage, reason string, retryAt time.Time) error
	// Reclaim returns messages whose lease expired (e.g. a crashed relay) to the
	// pending set. It returns the number of reclaimed rows.
	Reclaim(ctx context.Context, now time.Time) (int64, error)
}
