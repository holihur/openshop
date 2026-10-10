package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// The reconciliation checks are only useful if they actually catch drift, so the
// test creates it deliberately and asserts that it is found.
func TestReconciliationDetectsWalletDrift(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewReconciliationRepository(db)

	userID, walletID := testUUID(t), testUUID(t)
	now := time.Now().UTC()
	exec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
	             VALUES (?, ?, 'x', 'customer', 'active', ?, ?)`, userID, "drift-"+userID+"@example.com", now, now)
	exec(t, db, `INSERT INTO wallets (id, user_id, currency, balance_cents, created_at, updated_at)
	             VALUES (?, ?, 'CNY', 5000, ?, ?)`, walletID, userID, now, now)
	t.Cleanup(func() {
		exec(t, db, `DELETE FROM users WHERE id = ?`, userID)
	})

	// A consistent wallet is not reported.
	exec(t, db, `INSERT INTO wallet_transactions
	             (id, wallet_id, user_id, type, amount_cents, balance_after, description, created_at)
	             VALUES (?, ?, ?, 'topup', 5000, 5000, 'Top-up', ?)`,
		testUUID(t), walletID, userID, now)
	drift, err := repo.WalletDrift(ctx, 50)
	if err != nil {
		t.Fatalf("wallet drift: %v", err)
	}
	for _, row := range drift {
		if row.ID == walletID {
			t.Fatalf("a consistent wallet must not be reported: %+v", row)
		}
	}

	// The stored balance no longer matches the ledger.
	exec(t, db, `UPDATE wallets SET balance_cents = 6000 WHERE id = ?`, walletID)
	drift, err = repo.WalletDrift(ctx, 50)
	if err != nil {
		t.Fatalf("wallet drift: %v", err)
	}
	found := false
	for _, row := range drift {
		if row.ID != walletID {
			continue
		}
		found = true
		if row.Expected != 5000 || row.Stored != 6000 || row.Diff() != 1000 {
			t.Errorf("drift = %+v, want stored 6000 over ledger 5000", row)
		}
		if row.Unit != "CNY" || row.OwnerID != userID {
			t.Errorf("drift identity = %+v", row)
		}
	}
	if !found {
		t.Fatal("the drifted wallet was not reported")
	}
}

func TestReconciliationDetectsPointsDrift(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewReconciliationRepository(db)

	userID, accountID := testUUID(t), testUUID(t)
	now := time.Now().UTC()
	exec(t, db, `INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
	             VALUES (?, ?, 'x', 'customer', 'active', ?, ?)`, userID, "points-"+userID+"@example.com", now, now)
	exec(t, db, `INSERT INTO points_accounts (id, user_id, balance, lifetime_earned, created_at, updated_at)
	             VALUES (?, ?, 300, 300, ?, ?)`, accountID, userID, now, now)
	exec(t, db, `INSERT INTO points_transactions
	             (id, user_id, type, points, balance_after, description, created_at)
	             VALUES (?, ?, 'earn', 300, 300, 'Order', ?)`,
		testUUID(t), userID, now)
	t.Cleanup(func() {
		exec(t, db, `DELETE FROM users WHERE id = ?`, userID)
	})

	drift, err := repo.PointsDrift(ctx, 50)
	if err != nil {
		t.Fatalf("points drift: %v", err)
	}
	for _, row := range drift {
		if row.ID == accountID {
			t.Fatal("a consistent points account must not be reported")
		}
	}

	exec(t, db, `UPDATE points_accounts SET balance = 250 WHERE id = ?`, accountID)
	drift, err = repo.PointsDrift(ctx, 50)
	if err != nil {
		t.Fatalf("points drift: %v", err)
	}
	found := false
	for _, row := range drift {
		if row.ID == accountID {
			found = true
			if row.Unit != "points" || row.Diff() != -50 {
				t.Errorf("drift = %+v, want 50 points short", row)
			}
		}
	}
	if !found {
		t.Fatal("the drifted points account was not reported")
	}
}

func TestReconciliationDetectsUncollectedOrder(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewReconciliationRepository(db)

	orderID := testUUID(t)
	orderNo := "OSDRIFT" + orderID[:8]
	now := time.Now().UTC()
	exec(t, db, `INSERT INTO orders
	             (id, order_no, status, currency, subtotal_cents, total_cents, tax_cents,
	              shipping_cents, discount_cents, wallet_cents, points_discount_cents,
	              expires_at, created_at, updated_at)
	             VALUES (?, ?, 'paid', 'CNY', 10000, 10000, 0, 0, 0, 0, 0, ?, ?, ?)`,
		orderID, orderNo, now.Add(time.Hour), now, now)
	t.Cleanup(func() {
		exec(t, db, `DELETE FROM orders WHERE id = ?`, orderID)
	})

	// Paid, but with nothing collected: the order claims money that never
	// arrived, which is the case an operator must see.
	drift, err := repo.OrderDrift(ctx, 50)
	if err != nil {
		t.Fatalf("order drift: %v", err)
	}
	found := false
	for _, row := range drift {
		if row.ID != orderID {
			continue
		}
		found = true
		if row.ExpectedCents != 10000 || row.ActualCents != 0 || row.Diff() != -10000 {
			t.Errorf("drift = %+v, want 10000 expected and nothing collected", row)
		}
		if row.OrderNo != orderNo {
			t.Errorf("orderNo = %q", row.OrderNo)
		}
	}
	if !found {
		t.Fatal("an order with no collected payment was not reported")
	}

	// Once the payment is recorded the order reconciles, and it keeps
	// reconciling after a refund: the money did arrive and is accounted for
	// separately.
	exec(t, db, `INSERT INTO payments
	             (id, order_id, provider, provider_ref, amount_cents, currency, status, created_at, updated_at)
	             VALUES (?, ?, 'mock', ?, 10000, 'CNY', 'succeeded', ?, ?)`,
		testUUID(t), orderID, "ref-"+orderID[:8], now, now)
	drift, err = repo.OrderDrift(ctx, 50)
	if err != nil {
		t.Fatalf("order drift: %v", err)
	}
	for _, row := range drift {
		if row.ID == orderID {
			t.Fatalf("a settled order must not be reported: %+v", row)
		}
	}

	exec(t, db, `UPDATE payments SET status = 'refunded' WHERE order_id = ?`, orderID)
	exec(t, db, `UPDATE orders SET refunded_cents = 10000, status = 'refunded' WHERE id = ?`, orderID)
	exec(t, db, `INSERT INTO refunds (id, order_id, amount_cents, reason, restock, created_at)
	             VALUES (?, ?, 10000, 'test', false, ?)`, testUUID(t), orderID, now)
	drift, err = repo.OrderDrift(ctx, 50)
	if err != nil {
		t.Fatalf("order drift: %v", err)
	}
	for _, row := range drift {
		if row.ID == orderID {
			t.Fatalf("a refunded order must still reconcile: %+v", row)
		}
	}
}

func TestReconciliationIgnoresUnpaidOrders(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewReconciliationRepository(db)

	// An order still awaiting payment has collected nothing on purpose.
	orderID := testUUID(t)
	now := time.Now().UTC()
	exec(t, db, `INSERT INTO orders
	             (id, order_no, status, currency, subtotal_cents, total_cents, tax_cents,
	              shipping_cents, discount_cents, wallet_cents, points_discount_cents,
	              expires_at, created_at, updated_at)
	             VALUES (?, ?, 'pending_payment', 'CNY', 5000, 5000, 0, 0, 0, 0, 0, ?, ?, ?)`,
		orderID, "OSUNPAID"+orderID[:8], now.Add(time.Hour), now, now)
	t.Cleanup(func() {
		exec(t, db, `DELETE FROM orders WHERE id = ?`, orderID)
	})

	drift, err := repo.OrderDrift(ctx, 200)
	if err != nil {
		t.Fatalf("order drift: %v", err)
	}
	for _, row := range drift {
		if row.ID == orderID {
			t.Fatal("an order awaiting payment must not be reported as drift")
		}
	}
}

// exec runs a statement and fails the test if it errors, keeping the fixtures
// readable.
func exec(t *testing.T, db *DB, query string, args ...any) {
	t.Helper()
	if err := db.session(context.Background()).Exec(query, args...).Error; err != nil {
		t.Fatalf("exec %s: %v", query, err)
	}
}

var _ = domain.BalanceDrift{}
