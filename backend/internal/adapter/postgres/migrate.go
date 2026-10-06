package postgres

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// Migration is a versioned, ordered SQL change. DownSQL, when present, reverts
// it for `migrate -down`.
type Migration struct {
	Version string
	SQL     string
	DownSQL string
}

// advisoryLockKey is an arbitrary constant used with PostgreSQL advisory locks
// so that concurrent migration runs (e.g. during a rolling deploy) serialise
// instead of racing.
const advisoryLockKey int64 = 827349274

// Migrate applies pending migrations idempotently. It is safe to run from every
// replica at boot because the advisory lock makes only one runner proceed at a
// time; the rest wait, then observe the versions already recorded.
func (d *DB) Migrate(ctx context.Context, migrations []Migration) error {
	if err := d.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	// Acquire the session-level advisory lock and hold it for the whole run.
	if err := d.gorm.WithContext(ctx).Exec("SELECT pg_advisory_lock(?)", advisoryLockKey).Error; err != nil {
		return fmt.Errorf("migrate: acquire lock: %w", err)
	}
	defer func() {
		_ = d.gorm.WithContext(ctx).Exec("SELECT pg_advisory_unlock(?)", advisoryLockKey).Error
	}()

	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := d.applyOne(ctx, m); err != nil {
			return err
		}
		applied[m.Version] = true
	}
	return nil
}

func (d *DB) ensureMigrationsTable(ctx context.Context) error {
	return d.gorm.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`).Error
}

func (d *DB) appliedVersions(ctx context.Context) (map[string]bool, error) {
	var versions []string
	if err := d.gorm.WithContext(ctx).Raw("SELECT version FROM schema_migrations").Scan(&versions).Error; err != nil {
		return nil, fmt.Errorf("migrate: read versions: %w", err)
	}
	out := make(map[string]bool, len(versions))
	for _, v := range versions {
		out[v] = true
	}
	return out, nil
}

func (d *DB) applyOne(ctx context.Context, m Migration) error {
	return d.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(m.SQL).Error; err != nil {
			return fmt.Errorf("migrate %s: %w", m.Version, err)
		}
		return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version).Error
	})
}

// Rollback reverts the most recently applied migrations (up to steps) in
// reverse order. It takes the same advisory lock as Migrate so a rollback can
// never race a concurrent deploy. It returns how many migrations were reverted.
func (d *DB) Rollback(ctx context.Context, migrations []Migration, steps int) (int, error) {
	if steps <= 0 {
		return 0, nil
	}
	if err := d.ensureMigrationsTable(ctx); err != nil {
		return 0, err
	}
	if err := d.gorm.WithContext(ctx).Exec("SELECT pg_advisory_lock(?)", advisoryLockKey).Error; err != nil {
		return 0, fmt.Errorf("rollback: acquire lock: %w", err)
	}
	defer func() {
		_ = d.gorm.WithContext(ctx).Exec("SELECT pg_advisory_unlock(?)", advisoryLockKey).Error
	}()

	applied, err := d.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	reverted := 0
	for i := len(migrations) - 1; i >= 0 && reverted < steps; i-- {
		m := migrations[i]
		if !applied[m.Version] {
			continue
		}
		if strings.TrimSpace(m.DownSQL) == "" {
			return reverted, fmt.Errorf("rollback %s: no down migration found", m.Version)
		}
		if err := d.revertOne(ctx, m); err != nil {
			return reverted, err
		}
		reverted++
	}
	return reverted, nil
}

func (d *DB) revertOne(ctx context.Context, m Migration) error {
	return d.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(m.DownSQL).Error; err != nil {
			return fmt.Errorf("rollback %s: %w", m.Version, err)
		}
		return tx.Exec("DELETE FROM schema_migrations WHERE version = ?", m.Version).Error
	})
}
