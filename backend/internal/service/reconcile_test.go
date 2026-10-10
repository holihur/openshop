package service

import (
	"context"
	"errors"
	"testing"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

type fakeReconcileRepo struct {
	wallets []domain.BalanceDrift
	points  []domain.BalanceDrift
	orders  []domain.OrderDrift
	err     error
}

func (r *fakeReconcileRepo) WalletDrift(context.Context, int) ([]domain.BalanceDrift, error) {
	return r.wallets, r.err
}

func (r *fakeReconcileRepo) PointsDrift(context.Context, int) ([]domain.BalanceDrift, error) {
	return r.points, r.err
}

func (r *fakeReconcileRepo) OrderDrift(context.Context, int) ([]domain.OrderDrift, error) {
	return r.orders, r.err
}

// fakeAuditRepo captures what the reconciliation would have written.
type fakeAuditRepo struct {
	entries []domain.AuditLog
}

func (r *fakeAuditRepo) Create(_ context.Context, entry *domain.AuditLog) error {
	cp := *entry
	r.entries = append(r.entries, cp)
	return nil
}

func (r *fakeAuditRepo) List(context.Context, domain.AuditFilter) (domain.Page[domain.AuditLog], error) {
	return domain.Page[domain.AuditLog]{Items: r.entries}, nil
}

func (r *fakeAuditRepo) VerifyChain(context.Context) (int64, string, error) { return 0, "", nil }

type recordingMetrics struct {
	gauges map[string]float64
}

func (m *recordingMetrics) Counter(string, float64, map[string]string)   {}
func (m *recordingMetrics) Histogram(string, float64, map[string]string) {}
func (m *recordingMetrics) Gauge(name string, value float64, labels map[string]string) {
	if m.gauges == nil {
		m.gauges = map[string]float64{}
	}
	kind := labels["kind"]
	m.gauges[name+":"+kind] = value
}

func TestReconciliationReportsCleanState(t *testing.T) {
	metrics := &recordingMetrics{}
	svc := NewReconciliationService(&fakeReconcileRepo{}, nil, metrics, nil, 10)

	report, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !report.Clean() || report.Mismatches() != 0 {
		t.Fatalf("report = %+v, want clean", report)
	}
	// Every gauge is published even when clean, so a stale value cannot claim a
	// healthy shop after a mismatch has been fixed.
	for _, kind := range []string{"wallet", "points", "orders"} {
		if _, ok := metrics.gauges["openshop_reconciliation_mismatches:"+kind]; !ok {
			t.Errorf("gauge for %s was not published", kind)
		}
	}
}

func TestReconciliationCountsAndAuditsDrift(t *testing.T) {
	repo := &fakeReconcileRepo{
		wallets: []domain.BalanceDrift{{ID: "w1", OwnerID: "u1", Unit: "CNY", Stored: 6000, Expected: 5000}},
		orders:  []domain.OrderDrift{{ID: "o1", OrderNo: "OS1", ExpectedCents: 10000, ActualCents: 0}},
	}
	metrics := &recordingMetrics{}
	audit := NewAuditService(&fakeAuditRepo{}, &seqIDs{}, fixedClock{}, nil)
	svc := NewReconciliationService(repo, audit, metrics, nil, 10)

	report, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if report.Mismatches() != 2 || report.Clean() {
		t.Fatalf("report = %+v, want two mismatches", report)
	}
	if report.WalletDrift[0].Diff() != 1000 {
		t.Errorf("wallet diff = %d, want 1000", report.WalletDrift[0].Diff())
	}
	if report.OrderDrift[0].Diff() != -10000 {
		t.Errorf("order diff = %d, want -10000", report.OrderDrift[0].Diff())
	}
	if metrics.gauges["openshop_reconciliation_mismatches:wallet"] != 1 {
		t.Errorf("wallet gauge = %v", metrics.gauges)
	}
	if metrics.gauges["openshop_reconciliation_mismatches:orders"] != 1 {
		t.Errorf("order gauge = %v", metrics.gauges)
	}
	if metrics.gauges["openshop_reconciliation_mismatches:points"] != 0 {
		t.Errorf("a clean ledger must report zero: %v", metrics.gauges)
	}
}

func TestReconciliationPropagatesRepositoryErrors(t *testing.T) {
	svc := NewReconciliationService(&fakeReconcileRepo{err: errors.New("database is down")}, nil, &port.NopMetrics{}, nil, 10)
	if _, err := svc.Check(context.Background()); err == nil {
		t.Fatal("a failing check must surface the error")
	}
	// Reading for the console must not swallow it either.
	if _, err := svc.Latest(context.Background()); err == nil {
		t.Fatal("Latest must surface the error")
	}
}

func TestReconciliationLatestDoesNotAudit(t *testing.T) {
	repo := &fakeReconcileRepo{
		wallets: []domain.BalanceDrift{{ID: "w1", Unit: "CNY", Stored: 1, Expected: 0}},
	}
	auditRepo := &fakeAuditRepo{}
	audit := NewAuditService(auditRepo, &seqIDs{}, fixedClock{}, nil)
	svc := NewReconciliationService(repo, audit, &port.NopMetrics{}, nil, 10)

	// The worker audits; the console reads. Otherwise every dashboard refresh
	// would append to the trail.
	if _, err := svc.Latest(context.Background()); err != nil {
		t.Fatalf("latest: %v", err)
	}
	if len(auditRepo.entries) != 0 {
		t.Fatalf("Latest wrote %d audit entries, want none", len(auditRepo.entries))
	}
	if _, err := svc.Check(context.Background()); err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(auditRepo.entries) == 0 {
		t.Fatal("Check must audit what it finds")
	}
}
