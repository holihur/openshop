package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// TestAuditChainDetectsTampering appends entries, verifies the chain, then
// mutates and deletes stored rows to prove the tampering is reported.
func TestAuditChainDetectsTampering(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewAuditRepository(db)

	// Start from an empty chain so the counts are deterministic.
	if err := db.session(ctx).Exec("DELETE FROM audit_logs").Error; err != nil {
		t.Fatalf("reset: %v", err)
	}

	base := time.Now().UTC()
	ids := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i)
		ids = append(ids, id)
		entry := &domain.AuditLog{
			ID: id, ActorRole: "admin", Action: fmt.Sprintf("test.action.%d", i),
			ResourceType: "test", ResourceID: id, IP: "127.0.0.1",
			Metadata:  map[string]string{"index": fmt.Sprint(i)},
			CreatedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		if entry.Hash == "" {
			t.Fatalf("entry %d was stored without a hash", i)
		}
	}

	checked, broken, err := repo.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if broken != "" {
		t.Fatalf("chain should be intact, but %s was reported broken", broken)
	}
	if checked != 3 {
		t.Fatalf("checked %d entries, want 3", checked)
	}

	// Altering a stored field must break the chain at that entry.
	if err := db.session(ctx).Exec(
		"UPDATE audit_logs SET action = 'tampered' WHERE id = ?", ids[1]).Error; err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if _, broken, _ = repo.VerifyChain(ctx); broken != ids[1] {
		t.Fatalf("expected %s to be reported broken, got %q", ids[1], broken)
	}

	// Deleting a middle entry must break the link of its successor.
	if err := db.session(ctx).Exec(
		"UPDATE audit_logs SET action = 'test.action.1' WHERE id = ?", ids[1]).Error; err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := db.session(ctx).Exec("DELETE FROM audit_logs WHERE id = ?", ids[1]).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, broken, _ = repo.VerifyChain(ctx); broken != ids[2] {
		t.Fatalf("expected %s to be reported broken after deletion, got %q", ids[2], broken)
	}
}
