// Package postgres contains the GORM-backed implementations of the repository
// ports. GORM and the SQL driver are imported only here; services depend on the
// port interfaces, which keeps the persistence technology replaceable.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
	applog "github.com/holihur/openshop/internal/port"
)

// DB owns the connection pool and hands out request-scoped sessions.
type DB struct {
	gorm *gorm.DB
}

// txKey is the context key used to propagate a transaction to repositories.
type txKey struct{}

// Open connects to PostgreSQL and configures the pool. The pool settings are
// per-replica; PostgreSQL itself is the shared source of truth, so adding
// replicas scales read throughput.
func Open(cfg config.PostgresConfig, log applog.Logger) (*DB, error) {
	gormLog := logger.Default.LogMode(logger.Warn)
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger:                 gormLog,
		SkipDefaultTransaction: true,
		NowFunc:                func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	if log != nil {
		log.Info("postgres connected", "maxOpenConns", cfg.MaxOpenConns)
	}
	return &DB{gorm: db}, nil
}

func (d *DB) session(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return d.gorm.WithContext(ctx)
}

// WithinTx runs fn in a transaction, exposing it to repositories via context.
func (d *DB) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// AutoMigrate is a convenience for development. Production uses the versioned
// SQL migrations under backend/migrations.
func (d *DB) AutoMigrate() error {
	if err := d.gorm.AutoMigrate(
		&userModel{}, &categoryModel{}, &productModel{},
		&orderModel{}, &orderItemModel{}, &paymentModel{}, &outboxModel{},
		&couponModel{}, &couponRedemptionModel{}, &reviewModel{}, &variantModel{}, &addressModel{},
	); err != nil {
		return err
	}
	// GORM cannot express a stored generated tsvector column, so add it here to
	// keep AutoMigrate (development) consistent with the SQL migrations.
	return d.gorm.Exec(`
		ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector
		GENERATED ALWAYS AS (
			to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(description, ''))
		) STORED`).Error
}

func (d *DB) Close() error {
	sqlDB, err := d.gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	sqlDB, err := d.gorm.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// translate maps driver errors to domain sentinels so upper layers never import
// GORM error values.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrConflict
	}
	// The pgx driver reports unique violations as a string; keep the check loose.
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
		return domain.ErrConflict
	}
	return err
}
