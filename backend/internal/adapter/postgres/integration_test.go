package postgres

import (
	"os"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/config"
)

// openTestDB opens the database named by TEST_DATABASE_URL and brings the schema
// up to date. The test is skipped when the variable is absent, so a plain
// `go test ./...` still runs the unit tests without a database.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	db, err := Open(config.PostgresConfig{
		DSN: dsn, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
