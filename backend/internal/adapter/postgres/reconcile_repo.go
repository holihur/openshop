package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
)

// ReconciliationRepository runs the money invariants as SQL.
//
// Each check compares a materialised balance against the ledger that produced
// it. They are read-only: reporting a discrepancy is safe, silently rewriting
// money is not, so the worker records what it finds and leaves the correction to
// an operator.
type ReconciliationRepository struct{ db *DB }

func NewReconciliationRepository(db *DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

// WalletDrift lists wallets whose stored balance differs from their ledger.
func (r *ReconciliationRepository) WalletDrift(ctx context.Context, limit int) ([]domain.BalanceDrift, error) {
	var rows []struct {
		ID       string
		UserID   string
		Currency string
		Stored   int64
		Ledger   int64
	}
	err := r.db.session(ctx).Raw(`
		SELECT w.id, w.user_id, w.currency,
		       w.balance_cents AS stored,
		       coalesce((SELECT sum(t.amount_cents) FROM wallet_transactions t WHERE t.wallet_id = w.id), 0) AS ledger
		FROM wallets w
		WHERE w.balance_cents <> coalesce(
			(SELECT sum(t.amount_cents) FROM wallet_transactions t WHERE t.wallet_id = w.id), 0)
		ORDER BY w.updated_at DESC
		LIMIT ?`, limit).Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	out := make([]domain.BalanceDrift, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.BalanceDrift{
			ID: row.ID, OwnerID: row.UserID, Unit: row.Currency,
			Stored: row.Stored, Expected: row.Ledger,
		})
	}
	return out, nil
}

// PointsDrift lists loyalty accounts whose balance differs from their ledger.
func (r *ReconciliationRepository) PointsDrift(ctx context.Context, limit int) ([]domain.BalanceDrift, error) {
	var rows []struct {
		ID     string
		UserID string
		Stored int64
		Ledger int64
	}
	err := r.db.session(ctx).Raw(`
		SELECT a.id, a.user_id,
		       a.balance AS stored,
		       coalesce((SELECT sum(t.points) FROM points_transactions t WHERE t.user_id = a.user_id), 0) AS ledger
		FROM points_accounts a
		WHERE a.balance <> coalesce(
			(SELECT sum(t.points) FROM points_transactions t WHERE t.user_id = a.user_id), 0)
		ORDER BY a.updated_at DESC
		LIMIT ?`, limit).Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	out := make([]domain.BalanceDrift, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.BalanceDrift{
			ID: row.ID, OwnerID: row.UserID, Unit: "points",
			Stored: row.Stored, Expected: row.Ledger,
		})
	}
	return out, nil
}

// OrderDrift lists paid orders whose settled payments do not add up to what the
// order says was charged and refunded.
func (r *ReconciliationRepository) OrderDrift(ctx context.Context, limit int) ([]domain.OrderDrift, error) {
	var rows []struct {
		ID         string
		OrderNo    string
		Total      int64
		Refunded   int64
		Succeeded  int64
		RefundedOK int64
	}
	// total_cents is what the gateway was asked to collect (the gross amount
	// minus whatever was paid from the wallet or discounted with points), so a
	// settled order must have collected at least that much, and never more than
	// that plus what was refunded.
	//
	// A refunded payment keeps its status, so "collected" counts both succeeded
	// and refunded payments: the money did arrive, and the refund is measured
	// separately against refunded_cents.
	//
	// The check is deliberately conservative: both bounds are unambiguous
	// errors, whereas demanding exact equality would flag legitimate partial
	// refund sequences.
	err := r.db.session(ctx).Raw(`
		SELECT o.id, o.order_no,
		       o.total_cents AS total,
		       o.refunded_cents AS refunded,
		       coalesce((SELECT sum(p.amount_cents) FROM payments p
		                 WHERE p.order_id = o.id AND p.status IN ('succeeded', 'refunded')), 0) AS succeeded,
		       coalesce((SELECT sum(rf.amount_cents) FROM refunds rf
		                 WHERE rf.order_id = o.id), 0) AS refunded_ok
		FROM orders o
		WHERE o.status IN ('paid', 'shipped', 'completed', 'refunded')
		  AND (coalesce((SELECT sum(p.amount_cents) FROM payments p
		                 WHERE p.order_id = o.id AND p.status IN ('succeeded', 'refunded')), 0) < o.total_cents
		       OR coalesce((SELECT sum(p.amount_cents) FROM payments p
		                    WHERE p.order_id = o.id AND p.status IN ('succeeded', 'refunded')), 0)
		          > o.total_cents + o.refunded_cents)
		ORDER BY o.created_at DESC
		LIMIT ?`, limit).Scan(&rows).Error
	if err != nil {
		return nil, translate(err)
	}
	out := make([]domain.OrderDrift, 0, len(rows))
	for _, row := range rows {
		// The expected cash is the order total minus whatever was paid from
		// stored value, which never moves through a payment gateway.
		out = append(out, domain.OrderDrift{
			ID: row.ID, OrderNo: row.OrderNo,
			ExpectedCents: row.Total - row.Refunded,
			ActualCents:   row.Succeeded - row.RefundedOK,
		})
	}
	return out, nil
}

var _ = context.Background
