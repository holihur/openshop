// Package bootstrap is the composition root. It is the single place that knows
// about concrete adapters and wires them into the ports the application
// depends on. Everything below this package is infrastructure-agnostic.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/holihur/openshop/internal/adapter/logger"
	"github.com/holihur/openshop/internal/adapter/mail"
	"github.com/holihur/openshop/internal/adapter/metrics"
	natsadapter "github.com/holihur/openshop/internal/adapter/nats"
	"github.com/holihur/openshop/internal/adapter/payment"
	"github.com/holihur/openshop/internal/adapter/postgres"
	redisadapter "github.com/holihur/openshop/internal/adapter/redis"
	"github.com/holihur/openshop/internal/adapter/security"
	"github.com/holihur/openshop/internal/adapter/sms"
	"github.com/holihur/openshop/internal/adapter/storage"
	"github.com/holihur/openshop/internal/adapter/tracing"
	"github.com/holihur/openshop/internal/config"
	apphttp "github.com/holihur/openshop/internal/http"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
	"github.com/holihur/openshop/internal/worker"
)

// App holds every long-lived resource so it can be shut down deterministically.
type App struct {
	cfg    *config.Config
	log    *logger.Slog
	server *http.Server

	db    *postgres.DB
	redis *redisadapter.Client
	bus   *natsadapter.Bus

	handlers *handler.Handler

	// worker management
	wg              sync.WaitGroup
	workerCancel    context.CancelFunc
	shutdownTracing func(context.Context) error
}

// New builds the full dependency graph and returns a ready-to-run App.
func New(ctx context.Context, cfg *config.Config) (*App, error) {
	log := logger.New(cfg.App.LogLevel, cfg.App.IsProduction(), cfg.App.Name, "instance", cfg.App.InstanceID)

	// --- infrastructure (each behind a port) ---
	db, err := postgres.Open(cfg.Postgres, log)
	if err != nil {
		return nil, err
	}
	if cfg.Postgres.AutoMigrate {
		if err := db.AutoMigrate(); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
	}

	rdb, err := redisadapter.NewClient(cfg.Redis)
	if err != nil {
		return nil, err
	}

	bus, err := natsadapter.NewBus(cfg.NATS, log)
	if err != nil {
		return nil, err
	}

	// --- secondary adapters ---
	cache := redisadapter.NewCache(rdb)
	locker := redisadapter.NewLocker(rdb)
	cartRepo := redisadapter.NewCartRepository(rdb, 30*24*time.Hour)

	hasher := security.NewBcryptHasher()
	tokens := security.NewJWTIssuer(cfg.JWT.Secret, cfg.JWT.Issuer)
	ids := security.NewUUIDGenerator()
	clock := port.SystemClock{}

	payments := payment.NewRegistry(cfg.Payment.DefaultProvider,
		payment.NewMock(cfg.Payment.MockReturnURL, cfg.JWT.Secret),
	)

	objectStore, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		return nil, err
	}
	mailer := mail.New(cfg.Mail, log)
	smsSender := sms.New(cfg.SMS, log)
	_ = smsSender // reserved for OTP / notification flows
	promMetrics := metrics.New()
	tracer, shutdownTracing, err := tracing.Setup(ctx, tracing.Config{
		ServiceName: cfg.App.Name, InstanceID: cfg.App.InstanceID,
		Endpoint: cfg.Tracing.Endpoint, Insecure: cfg.Tracing.Insecure, SampleRatio: cfg.Tracing.SampleRatio,
	})
	if err != nil {
		return nil, err
	}

	// --- repositories ---
	users := postgres.NewUserRepository(db)
	categories := postgres.NewCategoryRepository(db)
	products := postgres.NewProductRepository(db)
	orders := postgres.NewOrderRepository(db)
	paymentRepo := postgres.NewPaymentRepository(db)
	couponRepo := postgres.NewCouponRepository(db)
	reviewRepo := postgres.NewReviewRepository(db)
	variantRepo := postgres.NewVariantRepository(db)
	addressRepo := postgres.NewAddressRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	shippingRepo := postgres.NewShippingMethodRepository(db)
	zoneRepo := postgres.NewShippingZoneRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	wishlistRepo := postgres.NewWishlistRepository(db)
	currencyRepo := postgres.NewCurrencyRepository(db)
	outbox := postgres.NewOutboxRepository(db)

	// --- services ---
	authSvc := service.NewAuthService(users, hasher, tokens, cache, ids, clock, mailer, service.AuthConfig{
		AccessTTL: cfg.JWT.AccessTTL, RefreshTTL: cfg.JWT.RefreshTTL, ResetBaseURL: cfg.App.PasswordResetURL,
		VerifyBaseURL: cfg.App.VerifyEmailURL, RequireEmailVerification: cfg.App.RequireEmailVerification,
	})
	catalogSvc := service.NewCatalogService(categories, products, variantRepo, cache, ids, clock, cfg.App.Currency)
	cartSvc := service.NewCartService(cartRepo, products, variantRepo)
	couponSvc := service.NewCouponService(couponRepo, ids, clock)
	reviewSvc := service.NewReviewService(reviewRepo, products, cache, ids, clock)
	addressSvc := service.NewAddressService(addressRepo, ids, clock)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo, cache, cfg.App.Currency, cfg.App.LowStockThreshold)
	shippingSvc := service.NewShippingService(shippingRepo, zoneRepo, ids, clock)
	auditSvc := service.NewAuditService(auditRepo, ids, clock, log)
	wishlistSvc := service.NewWishlistService(wishlistRepo, products)
	accountSvc := service.NewAccountService(users, addressRepo, wishlistRepo, reviewRepo, orders, authSvc, log)
	currencySvc := service.NewCurrencyService(currencyRepo, cfg.App.Currency, cache)
	orderSvc := service.NewOrderService(orders, products, couponRepo, variantRepo, addressRepo, shippingRepo, zoneRepo, currencySvc, cartRepo, locker, db, outbox, ids, clock, log, catalogSvc, promMetrics, tracer, cfg.App.TaxRateBps, cfg.App.OrderTTL, cfg.App.Currency)
	paymentSvc := service.NewPaymentService(paymentRepo, orders, payments, orderSvc, ids, clock, log, promMetrics)

	// --- HTTP surface ---
	h := &handler.Handler{
		Auth: authSvc, Catalog: catalogSvc, Cart: cartSvc, Orders: orderSvc,
		Payments: paymentSvc, Coupons: couponSvc, Reviews: reviewSvc, Addresses: addressSvc, Analytics: analyticsSvc, Shipping: shippingSvc, Audit: auditSvc, Wishlist: wishlistSvc, Currency: currencySvc, Account: accountSvc,
		Storage: objectStore, IDs: ids, Logger: log,
		SiteURL: cfg.App.PublicSiteURL,
		Metrics: promMetrics.Handler(),
		Checks: []handler.ReadinessCheck{
			{Name: "postgres", Check: db.Ping},
			{Name: "redis", Check: rdb.Ping},
			{Name: "nats", Check: func(context.Context) error { return bus.Ready() }},
		},
	}
	router := apphttp.NewRouter(cfg, tokens, authSvc, cache, promMetrics, tracer, h)

	app := &App{
		cfg: cfg, log: log, db: db, redis: rdb, bus: bus, handlers: h,
		shutdownTracing: shutdownTracing,
		server: &http.Server{
			Addr:         cfg.HTTP.Addr,
			Handler:      router,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		},
	}

	// --- event consumers ---
	consumers := worker.NewConsumers(bus, catalogSvc, mailer, log, tracer)
	if err := consumers.Start(); err != nil {
		app.Close(context.Background())
		return nil, err
	}

	// --- scheduled jobs ---
	workerCtx, workerCancel := context.WithCancel(context.Background())
	app.workerCancel = workerCancel
	if cfg.Worker.Enabled {
		sweeper := worker.NewOrderSweeper(orderSvc, catalogSvc, locker, clock, log,
			cfg.Worker.OrderSweepInterval, cfg.Worker.OrderSweepBatch, cfg.Worker.LeaderLockTTL)
		app.wg.Add(1)
		go func() {
			defer app.wg.Done()
			app.runWorker(workerCtx, "order-sweeper", sweeper.Run)
		}()

		// The outbox relay is safe to run on every replica; Claim uses SKIP LOCKED.
		relay := worker.NewOutboxRelay(outbox, bus, clock, log, promMetrics, tracer,
			cfg.Worker.OutboxInterval, cfg.Worker.OutboxBatch)
		app.wg.Add(1)
		go func() {
			defer app.wg.Done()
			app.runWorker(workerCtx, "outbox-relay", relay.Run)
		}()
	}

	log.Info("application initialised",
		"env", cfg.App.Env, "currency", cfg.App.Currency,
		"storage", cfg.Storage.Driver, "paymentProvider", cfg.Payment.DefaultProvider,
		"workers", cfg.Worker.Enabled)
	return app, nil
}

// runWorker wraps a job with logging and panic protection. The context is
// cancelled during shutdown so jobs exit promptly.
func (a *App) runWorker(ctx context.Context, name string, run func(ctx context.Context)) {
	a.log.Info("worker started", "worker", name)
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("worker panicked", "worker", name, "error", r)
		}
		a.log.Info("worker stopped", "worker", name)
	}()
	run(ctx)
}

// Run starts the HTTP server and blocks until the context is cancelled or the
// server fails, then shuts everything down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("http server listening", "addr", a.cfg.HTTP.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		a.log.Error("http server error", "error", err)
		return err
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	// Stop accepting new requests first, then drain the rest.
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http server shutdown error", "error", err)
	}
	if a.workerCancel != nil {
		a.workerCancel()
	}
	a.wg.Wait()
	a.Close(shutdownCtx)
	a.log.Info("shutdown complete")
	return nil
}

// Close releases infrastructure resources. It is safe to call more than once.
func (a *App) Close(ctx context.Context) {
	if a.shutdownTracing != nil {
		_ = a.shutdownTracing(ctx)
	}
	if a.bus != nil {
		_ = a.bus.Close()
	}
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}
