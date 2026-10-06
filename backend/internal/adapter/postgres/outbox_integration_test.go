package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/port"
)

// TestOutboxClaimSemantics exercises the transactional outbox against a real
// PostgreSQL instance. It is skipped unless TEST_DATABASE_URL is set, e.g.:
//
//	TEST_DATABASE_URL='host=localhost user=openshop password=openshop dbname=openshop sslmode=disable' go test ./internal/adapter/postgres/
func TestOutboxClaimSemantics(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	ctx := context.Background()
	db, err := Open(config.PostgresConfig{
		DSN: dsn, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewOutboxRepository(db)
	prefix := fmt.Sprintf("itest.%d", time.Now().UnixNano())
	defer func() {
		_ = db.gorm.WithContext(ctx).Exec("DELETE FROM outbox_events WHERE subject LIKE ?", prefix+"%").Error
	}()

	// Enqueue three events.
	for i := 0; i < 3; i++ {
		evt := port.Event{
			ID:      uuid.NewString(),
			Subject: fmt.Sprintf("%s.%d", prefix, i),
			Payload: []byte(`{"n":1}`),
		}
		if err := repo.Enqueue(ctx, evt); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	// Claim two; both become processing.
	claimed, err := repo.Claim(ctx, 2, 30*time.Second)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed = %d, want 2", len(claimed))
	}

	// A second claim while the first two are leased returns the third only.
	more, err := repo.Claim(ctx, 2, 30*time.Second)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(more) != 1 {
		t.Fatalf("second claim = %d, want 1 (SKIP LOCKED)", len(more))
	}

	// Publish one, leave the other leased.
	if err := repo.MarkPublished(ctx, claimed[0].ID); err != nil {
		t.Fatalf("mark published: %v", err)
	}

	// Reclaim after the lease expires: the two still-processing rows return to
	// pending (the third was published and must stay out of the queue).
	reclaimed, err := repo.Reclaim(ctx, time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if reclaimed != 2 {
		t.Fatalf("reclaimed = %d, want 2", reclaimed)
	}

	// Nothing else pending besides the reclaimed row should be claimable.
	remaining, err := repo.Claim(ctx, 10, 30*time.Second)
	if err != nil {
		t.Fatalf("final claim: %v", err)
	}
	for _, m := range remaining {
		if m.ID == claimed[0].ID {
			t.Fatalf("published message %s was claimed again", m.ID)
		}
	}
}
