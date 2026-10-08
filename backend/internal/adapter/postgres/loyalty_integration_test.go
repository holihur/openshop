package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// TestWalletBalanceIsAtomicAndNeverOverdraws checks the single most important
// invariant of the stored-value ledger: the balance is the sum of the applied
// deltas, and a debit that would go negative is refused without side effects.
func TestWalletBalanceIsAtomicAndNeverOverdraws(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)

	repo := NewWalletRepository(db)
	if _, err := repo.Ensure(ctx, user.ID, "CNY"); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	balance, err := repo.AddBalance(ctx, user.ID, 1000, false)
	if err != nil || balance != 1000 {
		t.Fatalf("credit: balance=%d err=%v, want 1000", balance, err)
	}

	balance, err = repo.AddBalance(ctx, user.ID, -400, false)
	if err != nil || balance != 600 {
		t.Fatalf("debit: balance=%d err=%v, want 600", balance, err)
	}

	if _, err := repo.AddBalance(ctx, user.ID, -10000, false); !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("overdraft must be refused, got %v", err)
	}
	wallet, err := repo.FindByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if wallet.BalanceCents != 600 {
		t.Fatalf("a refused debit must not change the balance, got %d", wallet.BalanceCents)
	}

	// A signed adjustment (reversal) may push the balance negative.
	if balance, err = repo.AddBalance(ctx, user.ID, -700, true); err != nil || balance != -100 {
		t.Fatalf("allowed negative: balance=%d err=%v, want -100", balance, err)
	}
}

func TestWalletLedgerRecordsRunningBalance(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)

	repo := NewWalletRepository(db)
	wallet, err := repo.Ensure(ctx, user.ID, "CNY")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	now := time.Now().UTC()
	for i, delta := range []int64{500, -200} {
		balance, err := repo.AddBalance(ctx, user.ID, delta, false)
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
		if err := repo.AddTransaction(ctx, &domain.WalletTransaction{
			ID: testUUID(t), WalletID: wallet.ID,
			UserID: user.ID, Type: domain.WalletAdjustment, AmountCents: delta,
			BalanceAfter: balance, CreatedAt: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("ledger %d: %v", i, err)
		}
	}

	page, err := repo.ListTransactions(ctx, domain.WalletTxFilter{UserID: user.ID, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("ledger has %d entries, want 2", page.Total)
	}
	// Newest first: balance after the second movement.
	if page.Items[0].BalanceAfter != 300 {
		t.Fatalf("latest running balance = %d, want 300", page.Items[0].BalanceAfter)
	}
}

func TestPointsNeverGoNegative(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db)

	repo := NewPointsRepository(db)
	if _, err := repo.Ensure(ctx, user.ID); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	balance, err := repo.AddPoints(ctx, user.ID, 50)
	if err != nil || balance != 50 {
		t.Fatalf("earn: balance=%d err=%v, want 50", balance, err)
	}
	balance, err = repo.AddPoints(ctx, user.ID, -20)
	if err != nil || balance != 30 {
		t.Fatalf("redeem: balance=%d err=%v, want 30", balance, err)
	}
	if _, err := repo.AddPoints(ctx, user.ID, -100); !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("over-redeem must be refused, got %v", err)
	}
	account, err := repo.FindByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if account.Balance != 30 {
		t.Fatalf("balance = %d, want 30", account.Balance)
	}
	// Only earning counts toward the lifetime total.
	if account.LifetimeEarned != 50 {
		t.Fatalf("lifetime earned = %d, want 50", account.LifetimeEarned)
	}
}

// createTestUser inserts a throwaway customer and removes it (with everything
// that cascades from it) when the test ends.
func createTestUser(t *testing.T, db *DB) *domain.User {
	t.Helper()
	ctx := context.Background()
	repo := NewUserRepository(db)
	now := time.Now().UTC()
	id := testUUID(t)
	user := &domain.User{
		ID: id, Email: "test-" + id + "@example.com", Name: "Test User",
		Role: domain.RoleCustomer, Status: domain.UserActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		_ = db.session(context.Background()).Exec("DELETE FROM users WHERE id = ?", id).Error
	})
	return user
}

// testUUID returns a random RFC 4122 version 4 identifier.
func testUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
