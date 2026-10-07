package worker

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/port"
)

// commissionSettler is implemented by service.CommissionService. Declaring it
// here keeps the worker free of a dependency on the service package.
type commissionSettler interface {
	SettleDue(ctx context.Context, limit int) (int, error)
}

// CommissionSettler pays out referral commissions whose cooling-off period has
// elapsed. It is leader-locked so only one replica runs the batch, and each
// commission is claimed by an atomic status transition, so duplicates are
// impossible even across restarts.
type CommissionSettler struct {
	settler  commissionSettler
	locker   port.Locker
	logger   port.Logger
	interval time.Duration
	batch    int
	lockTTL  time.Duration
}

func NewCommissionSettler(settler commissionSettler, locker port.Locker, logger port.Logger, interval time.Duration, batch int) *CommissionSettler {
	return &CommissionSettler{
		settler: settler, locker: locker, logger: logger,
		interval: interval, batch: batch, lockTTL: 30 * time.Second,
	}
}

func (w *CommissionSettler) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.settleOnce(ctx)
		}
	}
}

func (w *CommissionSettler) settleOnce(ctx context.Context) {
	lock, err := w.locker.Acquire(ctx, "lock:worker:commission", w.lockTTL, 0)
	if err != nil {
		return // another replica is settling
	}
	defer func() { _ = lock.Release(ctx) }()

	settled, err := w.settler.SettleDue(ctx, w.batch)
	if err != nil {
		w.logger.Warn("commission settlement failed", "error", err)
		return
	}
	if settled > 0 {
		w.logger.Info("settled referral commissions", "count", settled)
	}
}
