package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
)

// TestCursorPagination walks the product list with keyset pagination against a
// real PostgreSQL instance and asserts pages never overlap. Skipped unless
// TEST_DATABASE_URL is set.
func TestCursorPagination(t *testing.T) {
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

	repo := NewProductRepository(db)
	seen := map[string]bool{}
	cursor := ""
	pages := 0

	for {
		page, err := repo.List(ctx, domain.ProductFilter{CursorMode: true, Cursor: cursor, PageSize: 3})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, p := range page.Items {
			if seen[p.ID] {
				t.Fatalf("product %s appeared on more than one page", p.ID)
			}
			seen[p.ID] = true
		}
		pages++
		if page.NextCursor == "" || pages > 50 {
			break
		}
		cursor = page.NextCursor
	}
	if pages == 0 {
		t.Fatal("no pages returned")
	}
	t.Logf("walked %d product(s) over %d page(s)", len(seen), pages)
}
