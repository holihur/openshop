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

// withStatementCacheDisabled appends pgx parameters that turn off the prepared
// statement cache and describe each statement without caching. This avoids
// "cached plan must not change result type" errors, which GORM's SELECT * can
// raise after a migration alters a table, while keeping type-aware parameter
// encoding (unlike the simple protocol, which would encode []byte as bytea and
// break jsonb columns).
func withStatementCacheDisabled(dsn string) string {
	if strings.Contains(dsn, "default_query_exec_mode") {
		return dsn
	}
	const params = "statement_cache_capacity=0&default_query_exec_mode=describe_exec"
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		return dsn + sep + params
	}
	return strings.TrimSpace(dsn) + " " + strings.ReplaceAll(params, "&", " ")
}

// Open connects to PostgreSQL and configures the pool. The pool settings are
// per-replica; PostgreSQL itself is the shared source of truth, so adding
// replicas scales read throughput.
func Open(cfg config.PostgresConfig, log applog.Logger) (*DB, error) {
	gormLog := logger.Default.LogMode(logger.Warn)
	db, err := gorm.Open(postgres.Open(withStatementCacheDisabled(cfg.DSN)), &gorm.Config{
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
		&couponModel{}, &couponRedemptionModel{}, &reviewModel{}, &variantModel{}, &addressModel{}, &shippingMethodModel{}, &auditModel{}, &wishlistModel{}, &shippingZoneModel{}, &shippingRateModel{}, &refundModel{}, &returnModel{}, &settingModel{}, &productFAQModel{}, &ticketModel{}, &ticketMessageModel{},
		&walletModel{}, &walletTxModel{}, &pointsAccountModel{}, &pointsTxModel{},
		&referralCodeModel{}, &referralModel{}, &commissionModel{}, &withdrawalModel{}, &notificationModel{}, &personalAccessTokenModel{}, &socialAccountModel{},
	); err != nil {
		return err
	}
	if err := d.ensureSearchSchema(); err != nil {
		return err
	}
	// Ticket numbers come from a sequence declared in the SQL migrations; create
	// it here too so the AutoMigrate path can insert tickets.
	if err := d.gorm.Exec(`CREATE SEQUENCE IF NOT EXISTS ticket_number_seq START 1000`).Error; err != nil {
		return err
	}
	// The audit hash chain is ordered by a sequence that the model deliberately
	// omits (so an insert never writes it), which GORM therefore cannot create.
	// Without it the chain cannot be appended to or verified.
	if err := d.gorm.Exec(`ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS seq BIGSERIAL`).Error; err != nil {
		return err
	}
	return d.gorm.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_seq ON audit_logs (seq)`).Error
}

// ensureSearchSchema brings the search objects up to what the SQL migrations
// declare: GORM cannot express a stored generated tsvector column or a
// functional index, so they are created here. It is a repair, not just a
// creation: a database built by an older AutoMigrate still carries the
// unstemmed 'simple' vector, which would silently return worse results.
func (d *DB) ensureSearchSchema() error {
	if err := d.gorm.Exec(`CREATE EXTENSION IF NOT EXISTS pg_trgm`).Error; err != nil {
		return err
	}

	var expr string
	if err := d.gorm.Raw(`
		SELECT coalesce(pg_get_expr(ad.adbin, ad.adrelid), '')
		FROM pg_attribute a
		LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
		WHERE a.attrelid = 'products'::regclass AND a.attname = 'search_vector'`).Scan(&expr).Error; err != nil {
		return err
	}
	// Rebuild the column unless it already stems with the english dictionary.
	if !strings.Contains(expr, "english") {
		if err := d.gorm.Exec(`ALTER TABLE products DROP COLUMN IF EXISTS search_vector`).Error; err != nil {
			return err
		}
		if err := d.gorm.Exec(`
			ALTER TABLE products ADD COLUMN search_vector tsvector
			GENERATED ALWAYS AS (
				to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
			) STORED`).Error; err != nil {
			return err
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_products_search ON products USING GIN (search_vector)`,
		`CREATE INDEX IF NOT EXISTS idx_products_title_trgm ON products USING GIN (title gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS idx_products_price ON products (price_cents)`,
		`CREATE INDEX IF NOT EXISTS idx_variants_attributes ON product_variants USING GIN (attributes)`,
	}
	for _, stmt := range indexes {
		if err := d.gorm.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
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
