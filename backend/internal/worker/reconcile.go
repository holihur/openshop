package worker

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// Reconciler periodically verifies that the shop's stored money matches its
// ledgers. It is leader-locked so one replica checks per interval, and it only
// reports: the audit entry and the gauge are the product.
type Reconciler struct {
	reconciler *service.ReconciliationService
	locker     port.Locker
	logger     port.Logger
	interval   time.Duration
	lockTTL    time.Duration
}

func NewReconciler(reconciler *service.ReconciliationService, locker port.Locker, logger port.Logger, interval time.Duration) *Reconciler {
	return &Reconciler{
		reconciler: reconciler, locker: locker, logger: logger,
		interval: interval, lockTTL: 5 * time.Minute,
	}
}

func (w *Reconciler) Run(ctx context.Context) {
	// One check shortly after start, so a deployment does not wait a whole
	// interval to learn that something is wrong.
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		w.checkOnce(ctx)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkOnce(ctx)
		}
	}
}

func (w *Reconciler) checkOnce(ctx context.Context) {
	// The lock outlives a slow check, so a second replica cannot start a
	// concurrent run and double-report the same mismatches.
	lock, err := w.locker.Acquire(ctx, "lock:worker:reconcile", w.lockTTL, 0)
	if err != nil {
		return
	}
	defer func() { _ = lock.Release(ctx) }()

	report, err := w.reconciler.Check(ctx)
	if err != nil {
		w.logger.Warn("money reconciliation failed to run", "error", err)
		return
	}
	if report.Clean() {
		w.logger.Info("money reconciliation clean")
	}
}
