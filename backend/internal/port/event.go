package port

import "context"

// Event is a transport-agnostic message published to the event bus.
type Event struct {
	// Subject is the routing key, e.g. "order.created".
	Subject string
	// ID enables consumer-side idempotency.
	ID string
	// Payload is the JSON-encoded body.
	Payload []byte
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
