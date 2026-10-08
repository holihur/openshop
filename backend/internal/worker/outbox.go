package worker

import (
	"context"
	"math"
	"time"

	"github.com/holihur/openshop/internal/port"
)

// OutboxRelay publishes events persisted in the transactional outbox. Any
// number of replicas can run it safely: Claim uses SKIP LOCKED, so each message
// is leased by exactly one relay. Unlike the sweeper it needs no leader lock —
// more relays simply mean more throughput.
type OutboxRelay struct {
	outbox    port.Outbox
	inspector port.OutboxInspector
	bus       port.EventBus
	clock     port.Clock
	logger    port.Logger
	metrics   port.Metrics
	tracer    port.Tracer
	interval  time.Duration
	batch     int
	lease     time.Duration
	baseDelay time.Duration
	maxDelay  time.Duration
}

func NewOutboxRelay(
	outbox port.Outbox,
	bus port.EventBus,
	clock port.Clock,
	logger port.Logger,
	metrics port.Metrics,
	tracer port.Tracer,
	interval time.Duration,
	batch int,
) *OutboxRelay {
	if interval <= 0 {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 100
	}
	if tracer == nil {
		tracer = port.NoopTracer{}
	}
	return &OutboxRelay{
		outbox: outbox, bus: bus, clock: clock, logger: logger, metrics: metrics, tracer: tracer,
		interval: interval, batch: batch, lease: 30 * time.Second,
		baseDelay: time.Second, maxDelay: time.Minute,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.relayOnce(ctx)
			r.reportQueue(ctx)
		}
	}
}

// SetInspector wires the queue inspector so the relay can publish depth gauges.
func (r *OutboxRelay) SetInspector(i port.OutboxInspector) { r.inspector = i }

// reportQueue publishes the queue depth, the dead-letter count and the age of
// the oldest pending event. Without these an outage (or an event the relay gave
// up on) is invisible: the API stays healthy while events stop flowing.
func (r *OutboxRelay) reportQueue(ctx context.Context) {
	if r.inspector == nil || r.metrics == nil {
		return
	}
	stats, err := r.inspector.Stats(ctx)
	if err != nil {
		r.logger.Warn("outbox: stats failed", "error", err)
		return
	}
	r.metrics.Gauge("openshop_outbox_pending", float64(stats.Pending), nil)
	r.metrics.Gauge("openshop_outbox_processing", float64(stats.Processing), nil)
	r.metrics.Gauge("openshop_outbox_failed", float64(stats.Failed), nil)
	oldest := 0.0
	if stats.OldestPending != nil {
		oldest = time.Since(*stats.OldestPending).Seconds()
		if oldest < 0 {
			oldest = 0
		}
	}
	r.metrics.Gauge("openshop_outbox_oldest_pending_seconds", oldest, nil)
}

func (r *OutboxRelay) relayOnce(ctx context.Context) {
	now := r.clock.Now()

	if n, err := r.outbox.Reclaim(ctx, now); err != nil {
		r.logger.Error("outbox: reclaim failed", "error", err)
	} else if n > 0 {
		r.logger.Warn("outbox: reclaimed expired leases", "count", n)
	}

	msgs, err := r.outbox.Claim(ctx, r.batch, r.lease)
	if err != nil {
		r.logger.Error("outbox: claim failed", "error", err)
		return
	}
	for _, msg := range msgs {
		// Continue the trace that produced the event, so the async hop is visible
		// as part of one end-to-end trace.
		msgCtx := r.tracer.Extract(ctx, msg.TraceParent)
		msgCtx, span := r.tracer.Start(msgCtx, "outbox.publish "+msg.Subject,
			port.Attribute{Key: "messaging.system", Value: "nats"},
			port.Attribute{Key: "messaging.destination", Value: msg.Subject},
			port.Attribute{Key: "outbox.attempts", Value: msg.Attempts},
		)
		evt := port.Event{ID: msg.ID, Subject: msg.Subject, Payload: msg.Payload, TraceParent: r.tracer.Inject(msgCtx)}
		err := r.bus.Publish(msgCtx, evt)
		if err != nil {
			span.RecordError(err)
		}
		span.End()

		if err != nil {
			retryAt := now.Add(r.backoff(msg.Attempts))
			r.logger.Error("outbox: publish failed", "subject", msg.Subject, "id", msg.ID, "attempts", msg.Attempts, "error", err)
			if mErr := r.outbox.MarkFailed(ctx, msg, err.Error(), retryAt); mErr != nil {
				r.logger.Error("outbox: mark failed errored", "id", msg.ID, "error", mErr)
			}
			if r.metrics != nil {
				r.metrics.Counter("openshop_outbox_publish_failures_total", 1, map[string]string{"subject": msg.Subject})
			}
			continue
		}
		if err := r.outbox.MarkPublished(ctx, msg.ID); err != nil {
			r.logger.Error("outbox: mark published failed", "id", msg.ID, "error", err)
			continue
		}
		if r.metrics != nil {
			r.metrics.Counter("openshop_outbox_published_total", 1, map[string]string{"subject": msg.Subject})
		}
	}
}

// backoff grows exponentially with the attempt count, capped at maxDelay.
func (r *OutboxRelay) backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	d := time.Duration(float64(r.baseDelay) * math.Pow(2, float64(attempts-1)))
	if d > r.maxDelay || d <= 0 {
		d = r.maxDelay
	}
	return d
}
