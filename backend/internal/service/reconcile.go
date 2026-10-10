package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ReconciliationRepository is the read-only view of the money invariants.
type ReconciliationRepository interface {
	WalletDrift(ctx context.Context, limit int) ([]domain.BalanceDrift, error)
	PointsDrift(ctx context.Context, limit int) ([]domain.BalanceDrift, error)
	OrderDrift(ctx context.Context, limit int) ([]domain.OrderDrift, error)
}

// ReconciliationReport is what the checks found on one run.
type ReconciliationReport struct {
	// Checked lists how many rows each invariant examined, so a clean report can
	// be told apart from a check that did not run.
	WalletsChecked int
	PointsChecked  int
	OrdersChecked  int
	WalletDrift    []domain.BalanceDrift `json:"walletDrift"`
	PointsDrift    []domain.BalanceDrift `json:"pointsDrift"`
	OrderDrift     []domain.OrderDrift   `json:"orderDrift"`
}

// Clean reports whether every invariant held.
func (r ReconciliationReport) Clean() bool {
	return len(r.WalletDrift) == 0 && len(r.PointsDrift) == 0 && len(r.OrderDrift) == 0
}

// Mismatches totals the rows that disagree with their ledger.
func (r ReconciliationReport) Mismatches() int {
	return len(r.WalletDrift) + len(r.PointsDrift) + len(r.OrderDrift)
}

// ReconciliationService verifies that the money the shop believes it holds
// matches the ledgers that produced it.
//
// It never repairs what it finds: rewriting a balance automatically would hide
// the bug that caused the drift. Instead every mismatch is audited and counted,
// so it shows up on a dashboard and in the audit trail.
type ReconciliationService struct {
	repo    ReconciliationRepository
	audit   *AuditService
	metrics port.Metrics
	logger  port.Logger
	batch   int
}

func NewReconciliationService(repo ReconciliationRepository, audit *AuditService, metrics port.Metrics, logger port.Logger, batch int) *ReconciliationService {
	if metrics == nil {
		metrics = port.NopMetrics{}
	}
	if batch <= 0 {
		batch = 200
	}
	return &ReconciliationService{repo: repo, audit: audit, metrics: metrics, logger: logger, batch: batch}
}

// Check runs every invariant and publishes the result.
func (s *ReconciliationService) Check(ctx context.Context) (ReconciliationReport, error) {
	// Reported as 0 when clean, so a stale gauge cannot claim a healthy shop.
	s.metrics.Gauge("openshop_reconciliation_mismatches", 0, map[string]string{"kind": "wallet"})
	s.metrics.Gauge("openshop_reconciliation_mismatches", 0, map[string]string{"kind": "points"})
	s.metrics.Gauge("openshop_reconciliation_mismatches", 0, map[string]string{"kind": "orders"})

	report := ReconciliationReport{}
	var err error
	if report.WalletDrift, err = s.repo.WalletDrift(ctx, s.batch); err != nil {
		return report, err
	}
	if report.PointsDrift, err = s.repo.PointsDrift(ctx, s.batch); err != nil {
		return report, err
	}
	if report.OrderDrift, err = s.repo.OrderDrift(ctx, s.batch); err != nil {
		return report, err
	}
	report.WalletsChecked = len(report.WalletDrift)
	report.PointsChecked = len(report.PointsDrift)
	report.OrdersChecked = len(report.OrderDrift)

	s.metrics.Gauge("openshop_reconciliation_mismatches", float64(len(report.WalletDrift)), map[string]string{"kind": "wallet"})
	s.metrics.Gauge("openshop_reconciliation_mismatches", float64(len(report.PointsDrift)), map[string]string{"kind": "points"})
	s.metrics.Gauge("openshop_reconciliation_mismatches", float64(len(report.OrderDrift)), map[string]string{"kind": "orders"})

	if report.Clean() {
		return report, nil
	}
	if s.logger != nil {
		s.logger.Warn("money reconciliation found mismatches",
			"wallets", len(report.WalletDrift), "points", len(report.PointsDrift),
			"orders", len(report.OrderDrift))
	}
	// One audit entry per kind, with the first offending ids: a per-row entry
	// would flood the trail at exactly the moment it matters most.
	s.recordDrift(ctx, "reconciliation.wallet_drift", report.WalletDrift)
	s.recordDrift(ctx, "reconciliation.points_drift", report.PointsDrift)
	s.recordOrders(ctx, "reconciliation.order_drift", report.OrderDrift)
	return report, nil
}

// Latest re-runs the checks for the ops console, without touching the audit
// trail: reading the state is not an event.
func (s *ReconciliationService) Latest(ctx context.Context) (ReconciliationReport, error) {
	report := ReconciliationReport{}
	var err error
	if report.WalletDrift, err = s.repo.WalletDrift(ctx, s.batch); err != nil {
		return report, err
	}
	if report.PointsDrift, err = s.repo.PointsDrift(ctx, s.batch); err != nil {
		return report, err
	}
	if report.OrderDrift, err = s.repo.OrderDrift(ctx, s.batch); err != nil {
		return report, err
	}
	return report, nil
}

// recordOrders audits order mismatches, which have their own shape.
func (s *ReconciliationService) recordOrders(ctx context.Context, action string, drift []domain.OrderDrift) {
	if s.audit == nil || len(drift) == 0 {
		return
	}
	sample := make([]string, 0, 5)
	for i, row := range drift {
		if i == 5 {
			break
		}
		sample = append(sample, row.OrderNo)
	}
	s.audit.Record(ctx, Entry{
		Action: action, ResourceType: "reconciliation",
		Metadata: map[string]string{
			"count":     strconv.Itoa(len(drift)),
			"sample":    strings.Join(sample, ","),
			"expected":  strconv.FormatInt(drift[0].ExpectedCents, 10),
			"collected": strconv.FormatInt(drift[0].ActualCents, 10),
		},
	})
}

func (s *ReconciliationService) recordDrift(ctx context.Context, action string, drift []domain.BalanceDrift) {
	if s.audit == nil || len(drift) == 0 {
		return
	}
	sample := make([]string, 0, 5)
	for i, row := range drift {
		if i == 5 {
			break
		}
		sample = append(sample, row.ID)
	}
	s.audit.Record(ctx, Entry{
		Action: action, ResourceType: "reconciliation",
		Metadata: map[string]string{
			"count":    strconv.Itoa(len(drift)),
			"sample":   strings.Join(sample, ","),
			"expected": strconv.FormatInt(drift[0].Expected, 10),
			"stored":   strconv.FormatInt(drift[0].Stored, 10),
		},
	})
}
