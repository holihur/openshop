package worker

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/port"
)

// Retention prunes published outbox events and old audit logs in bounded
// batches so those append-only tables do not grow without limit. It is
// leader-locked, so only one replica prunes at a time even when many run.
type Retention struct {
	pruner    port.RetentionRepository
	locker    port.Locker
	logger    port.Logger
	interval  time.Duration
	batch     int
	outboxTTL time.Duration
	auditTTL  time.Duration
	lockTTL   time.Duration
}

func NewRetention(
	pruner port.RetentionRepository,
	locker port.Locker,
	logger port.Logger,
	interval time.Duration,
	batch int,
	outboxTTL, auditTTL time.Duration,
) *Retention {
	return &Retention{
		pruner: pruner, locker: locker, logger: logger,
		interval: interval, batch: batch, outboxTTL: outboxTTL, auditTTL: auditTTL,
		lockTTL: 30 * time.Second,
	}
}

func (w *Retention) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pruneOnce(ctx)
		}
	}
}

func (w *Retention) pruneOnce(ctx context.Context) {
	lock, err := w.locker.Acquire(ctx, "lock:worker:retention", w.lockTTL, 0)
	if err != nil {
		return // another replica is pruning
	}
	defer func() { _ = lock.Release(ctx) }()

	now := time.Now().UTC()
	if n, err := w.pruner.DeletePublishedOutboxBefore(ctx, now.Add(-w.outboxTTL), w.batch); err == nil && n > 0 {
		w.logger.Info("pruned outbox events", "rows", n)
	}
	if n, err := w.pruner.DeleteAuditBefore(ctx, now.Add(-w.auditTTL), w.batch); err == nil && n > 0 {
		w.logger.Info("pruned audit logs", "rows", n)
	}
}
