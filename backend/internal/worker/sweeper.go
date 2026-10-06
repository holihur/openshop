// Package worker contains background consumers and scheduled jobs. Every job is
// written so that running it on many replicas is safe: consumers use NATS queue
// groups (one delivery per message) and sweeps take a Redis leader lock.
package worker

import (
	"context"
	"errors"
	"time"

	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// OrderSweeper cancels orders whose payment window has elapsed and returns the
// reserved stock. A Redis leader lock guarantees a single replica sweeps per
// interval; if that replica dies the lock expires and another takes over.
type OrderSweeper struct {
	orders   *service.OrderService
	catalog  *service.CatalogService
	locker   port.Locker
	clock    port.Clock
	logger   port.Logger
	interval time.Duration
	batch    int
	lockTTL  time.Duration
}

func NewOrderSweeper(
	orders *service.OrderService,
	catalog *service.CatalogService,
	locker port.Locker,
	clock port.Clock,
	logger port.Logger,
	interval time.Duration,
	batch int,
	lockTTL time.Duration,
) *OrderSweeper {
	if interval <= 0 {
		interval = time.Minute
	}
	if batch <= 0 {
		batch = 100
	}
	if lockTTL <= 0 {
		lockTTL = interval * 2
	}
	return &OrderSweeper{
		orders: orders, catalog: catalog, locker: locker, clock: clock, logger: logger,
		interval: interval, batch: batch, lockTTL: lockTTL,
	}
}

// Run blocks until the context is cancelled, sweeping on every tick.
func (w *OrderSweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Sweep once on startup so a rolling restart does not wait a full interval.
	w.sweepOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sweepOnce(ctx)
		}
	}
}

func (w *OrderSweeper) sweepOnce(ctx context.Context) {
	// Non-blocking acquisition: if another replica leads, skip this tick.
	lock, err := w.locker.Acquire(ctx, "lock:worker:order-sweep", w.lockTTL, 0)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		// ErrLockUnavailable is expected when another instance is leading.
		return
	}
	defer func() { _ = lock.Release(ctx) }()

	now := w.clock.Now()
	expired, err := w.orders.ListExpired(ctx, now, w.batch)
	if err != nil {
		w.logger.Error("order sweeper: list expired failed", "error", err)
		return
	}
	if len(expired) == 0 {
		return
	}

	var cancelled int
	for _, o := range expired {
		if err := w.orders.CancelExpired(ctx, o.ID); err != nil {
			w.logger.Error("order sweeper: cancel failed", "orderId", o.ID, "error", err)
			continue
		}
		ids := make([]string, 0, len(o.Items))
		for _, it := range o.Items {
			ids = append(ids, it.ProductID)
		}
		w.catalog.InvalidateProductCache(ctx, ids...)
		cancelled++
	}
	w.logger.Info("order sweeper: expired orders cancelled", "count", cancelled, "scanned", len(expired))
}
