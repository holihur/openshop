// Package bootstrap is the composition root. It is the single place that knows
// about concrete adapters and wires them into the ports the application
// depends on. Everything below this package is infrastructure-agnostic.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/holihur/openshop/internal/adapter/invoice"
	"github.com/holihur/openshop/internal/adapter/logger"
	"github.com/holihur/openshop/internal/adapter/mail"
	"github.com/holihur/openshop/internal/adapter/metrics"
	natsadapter "github.com/holihur/openshop/internal/adapter/nats"
	"github.com/holihur/openshop/internal/adapter/oidc"
	"github.com/holihur/openshop/internal/adapter/payment"
	"github.com/holihur/openshop/internal/adapter/postgres"
	redisadapter "github.com/holihur/openshop/internal/adapter/redis"
	"github.com/holihur/openshop/internal/adapter/security"
	"github.com/holihur/openshop/internal/adapter/sms"
	"github.com/holihur/openshop/internal/adapter/storage"
	"github.com/holihur/openshop/internal/adapter/tracing"
	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
	"github.com/holihur/openshop/internal/version"
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

// Surface identifies which binary is being built, so only the services that
// surface needs are constructed.
type Surface string

const (
	SurfaceFront Surface = "front"
	SurfaceOps   Surface = "ops"
)

// allowedRole maps a surface to the role its sessions are restricted to: the
// storefront only signs in customers, ops only administrators.
// paymentProviders registers the payment adapters. Stripe is only registered
// when a secret key is configured; the key itself stays in the environment.
func paymentProviders(cfg *config.Config) []port.PaymentProvider {
	providers := []port.PaymentProvider{
		payment.NewMock(cfg.Payment.MockReturnURL, cfg.JWT.Secret),
		payment.NewOffline(),
	}
	if cfg.Payment.StripeSecretKey != "" {
		providers = append(providers, payment.NewStripe(cfg.Payment.StripeSecretKey, cfg.Payment.StripeWebhookSecret))
	}
	return providers
}

func allowedRoles(s Surface) []domain.UserRole {
	if s == SurfaceOps {
		return []domain.UserRole{domain.RoleAdmin, domain.RoleSupport, domain.RoleCatalog, domain.RoleFinance}
	}
	return []domain.UserRole{domain.RoleCustomer}
}

// HTTPDeps are the wired dependencies a surface package (internal/http/front
// or internal/http/ops) needs to build its engine. Injecting the builder keeps
// each binary free of the other surface: the linker drops whatever is unused.
type HTTPDeps struct {
	Config   *config.Config
	Handler  *handler.Handler
	Tokens   port.TokenIssuer
	Auth     *service.AuthService
	Cache    port.Cache
	Limiter  port.RateLimiter
	Metrics  port.Metrics
	Tracer   port.Tracer
	Settings *service.SettingsService

	// Surface-specific services (nil for the other surface).
	Cart      *service.CartService
	Addresses *service.AddressService
	Wishlist  *service.WishlistService
	Account   *service.AccountService
	Analytics *service.AnalyticsService
}

// Options selects the HTTP surface and whether background workers run. The
// storefront binary runs workers; the ops binary does not.
type Options struct {
	// BuildHTTP constructs the surface's HTTP handler. Required.
	BuildHTTP func(HTTPDeps) http.Handler
	// Surface selects which surface-specific services to construct.
	Surface Surface
	// RunWorkers starts the event consumers and scheduled jobs.
	RunWorkers bool
	// Addr overrides the listen address (defaults to cfg.HTTP.Addr).
	Addr string
}

// New builds the full dependency graph and returns a ready-to-run App.
func New(ctx context.Context, cfg *config.Config, opts Options) (*App, error) {
	log := logger.New(cfg.App.LogLevel, cfg.App.IsProduction(), cfg.App.Name, "instance", cfg.App.InstanceID)
	log.Info("starting", "version", version.Version, "commit", version.Commit)

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

	// Normalise the surface so audience scoping and service selection are always
	// well-defined; the storefront is the default.
	surface := opts.Surface
	if surface == "" {
		surface = SurfaceFront
	}

	// --- secondary adapters ---
	cache := redisadapter.NewCache(rdb)
	locker := redisadapter.NewLocker(rdb)
	limiter := redisadapter.NewRateLimiter(rdb)
	cartRepo := redisadapter.NewCartRepository(rdb, 30*24*time.Hour)

	hasher := security.NewBcryptHasher()
	// The audience scopes tokens to this binary: storefront tokens are rejected
	// by the ops binary and vice versa.
	tokens := security.NewJWTIssuer(cfg.JWT.Secret, cfg.JWT.Issuer, string(surface))
	ids := security.NewUUIDGenerator()
	clock := port.SystemClock{}

	payments := payment.NewRegistry(cfg.Payment.DefaultProvider, paymentProviders(cfg)...)

	objectStore, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		return nil, err
	}
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
	refundRepo := postgres.NewRefundRepository(db)
	returnRepo := postgres.NewReturnRepository(db)
	ticketRepo := postgres.NewTicketRepository(db)
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
	faqRepo := postgres.NewProductFAQRepository(db)
	outbox := postgres.NewOutboxRepository(db)

	// Runtime settings: defaults come from the environment, overrides from the
	// ops console. Everything here can change without a restart.
	settingsRepo := postgres.NewSettingsRepository(db)
	settingsSvc := service.NewSettingsService(settingsRepo, clock, map[string]string{
		"store.public_url":                cfg.App.PublicSiteURL,
		"checkout.tax_rate_bps":           strconv.Itoa(cfg.App.TaxRateBps),
		"checkout.order_ttl_minutes":      strconv.Itoa(int(cfg.App.OrderTTL.Minutes())),
		"inventory.low_stock_threshold":   strconv.Itoa(cfg.App.LowStockThreshold),
		"auth.require_email_verification": strconv.FormatBool(cfg.App.RequireEmailVerification),
		"auth.allow_registration":         strconv.FormatBool(cfg.App.AllowRegistration),
		"auth.password_reset_url":         cfg.App.PasswordResetURL,
		"auth.email_verify_url":           cfg.App.VerifyEmailURL,
		"security.rate_limit_rps":         strconv.Itoa(cfg.HTTP.RateLimitRPS),
		"security.rate_limit_user_rps":    strconv.Itoa(cfg.HTTP.RateLimitUserRPS),
		"security.auth_rate_limit_rps":    strconv.Itoa(cfg.HTTP.RateLimitAuthRPS),
		"payment.default_provider":        cfg.Payment.DefaultProvider,
		"mail.driver":                     cfg.Mail.Driver,
		"mail.from":                       cfg.Mail.From,
		"mail.host":                       cfg.Mail.Host,
		"mail.port":                       strconv.Itoa(cfg.Mail.Port),
		"mail.user":                       cfg.Mail.User,
	})
	// The mail transport is resolved from settings on every send.
	mailer := mail.NewDynamic(settingsSvc, cfg.Mail.Pass, log)
	oidcSvc := service.NewOIDCService(settingsSvc, cfg.OIDC.ClientSecret, cfg.OIDC.ClientSecrets,
		func(ctx context.Context, issuer, clientID, clientSecret, redirectURL string, scopes []string) (port.IdentityProvider, error) {
			return oidc.New(ctx, issuer, clientID, clientSecret, redirectURL, scopes)
		})

	// --- services ---
	authSvc := service.NewAuthService(users, hasher, tokens, cache, ids, clock, mailer, service.AuthConfig{
		AccessTTL: cfg.JWT.AccessTTL, RefreshTTL: cfg.JWT.RefreshTTL,
		AllowedRoles: allowedRoles(surface),
	}, settingsSvc)
	catalogSvc := service.NewCatalogService(categories, products, variantRepo, faqRepo, cache, ids, clock, cfg.App.Currency)
	couponSvc := service.NewCouponService(couponRepo, ids, clock)
	reviewSvc := service.NewReviewService(reviewRepo, products, orders, cache, ids, clock)
	shippingSvc := service.NewShippingService(shippingRepo, zoneRepo, ids, clock)
	auditSvc := service.NewAuditService(auditRepo, ids, clock, log)
	currencySvc := service.NewCurrencyService(currencyRepo, cfg.App.Currency, cache)
	customerSvc := service.NewCustomerService(users, clock)

	// Surface-specific services: only the binary that serves them builds them, so
	// e.g. the ops binary never constructs cart/wishlist/address services.
	var (
		cartSvc      *service.CartService
		addressSvc   *service.AddressService
		wishlistSvc  *service.WishlistService
		accountSvc   *service.AccountService
		analyticsSvc *service.AnalyticsService
	)
	if surface == SurfaceOps {
		analyticsSvc = service.NewAnalyticsService(analyticsRepo, cache, cfg.App.Currency, settingsSvc)
	} else {
		cartSvc = service.NewCartService(cartRepo, products, variantRepo)
		addressSvc = service.NewAddressService(addressRepo, ids, clock)
		wishlistSvc = service.NewWishlistService(wishlistRepo, products)
		accountSvc = service.NewAccountService(users, addressRepo, wishlistRepo, reviewRepo, orders, authSvc, log)
	}
	returnSvc := service.NewReturnService(returnRepo, orders, ids, clock)
	ticketSvc := service.NewTicketService(ticketRepo, users, orders, ids, clock, outbox, settingsSvc)
	retentionRepo := postgres.NewRetentionRepository(db)
	orderSvc := service.NewOrderService(orders, users, products, couponRepo, variantRepo, addressRepo, shippingRepo, zoneRepo, currencySvc, cartRepo, locker, db, outbox, ids, clock, log, catalogSvc, promMetrics, tracer, invoice.NewPDFRenderer(), settingsSvc, cfg.App.Currency)
	paymentSvc := service.NewPaymentService(paymentRepo, refundRepo, orders, payments, orderSvc, ids, clock, log, promMetrics, settingsSvc)

	// Wallet, loyalty points and referral commissions settle into the wallet.
	walletSvc := service.NewWalletService(postgres.NewWalletRepository(db), ids, clock, settingsSvc, cfg.App.Currency)
	pointsSvc := service.NewPointsService(postgres.NewPointsRepository(db), ids, clock, settingsSvc)
	commissionSvc := service.NewCommissionService(postgres.NewReferralRepository(db), postgres.NewCommissionRepository(db), walletSvc, ids, clock, settingsSvc)
	orderSvc.SetLoyalty(walletSvc, pointsSvc, commissionSvc)
	withdrawalSvc := service.NewWithdrawalService(postgres.NewWithdrawalRepository(db), walletSvc, ids, clock, db, settingsSvc, cfg.App.Currency)
	notificationSvc := service.NewNotificationService(postgres.NewNotificationRepository(db), ids, clock)
	patSvc := service.NewPATService(postgres.NewPersonalAccessTokenRepository(db), ids, clock)
	withdrawalSvc.SetNotifier(notificationSvc)
	commissionSvc.SetNotifier(notificationSvc)
	ticketSvc.SetNotifier(notificationSvc)

	// --- HTTP surface ---
	h := &handler.Handler{
		Auth: authSvc, Catalog: catalogSvc, Orders: orderSvc, Payments: paymentSvc,
		Coupons: couponSvc, Reviews: reviewSvc, Shipping: shippingSvc, Audit: auditSvc,
		Currency: currencySvc, Returns: returnSvc, Tickets: ticketSvc, Wallet: walletSvc, Points: pointsSvc, Commission: commissionSvc, Withdrawals: withdrawalSvc, Notifications: notificationSvc, PATs: patSvc, Outbox: outbox, Settings: settingsSvc, Customers: customerSvc, OIDC: oidcSvc, Storage: objectStore, Cache: cache, IDs: ids, Logger: log,
		Metrics: promMetrics.Handler(),
		Checks: []handler.ReadinessCheck{
			{Name: "postgres", Check: db.Ping},
			{Name: "redis", Check: rdb.Ping},
			{Name: "nats", Check: func(context.Context) error { return bus.Ready() }},
		},
	}
	if opts.BuildHTTP == nil {
		return nil, fmt.Errorf("bootstrap: Options.BuildHTTP is required")
	}
	addr := opts.Addr
	if addr == "" {
		addr = cfg.HTTP.Addr
	}
	engine := opts.BuildHTTP(HTTPDeps{
		Config: cfg, Handler: h, Tokens: tokens, Auth: authSvc,
		Cache: cache, Limiter: limiter, Metrics: promMetrics, Tracer: tracer, Settings: settingsSvc,
		Cart: cartSvc, Addresses: addressSvc, Wishlist: wishlistSvc, Account: accountSvc, Analytics: analyticsSvc,
	})

	app := &App{
		cfg: cfg, log: log, db: db, redis: rdb, bus: bus, handlers: h,
		shutdownTracing: shutdownTracing,
		server: &http.Server{
			Addr:         addr,
			Handler:      engine,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		},
	}

	// Event consumers and scheduled jobs run only on the storefront binary; the
	// ops binary is a pure admin surface and stays worker-free.
	if opts.RunWorkers {
		consumers := worker.NewConsumers(bus, catalogSvc, mailer, log, tracer, notificationSvc)
		if err := consumers.Start(); err != nil {
			app.Close(context.Background())
			return nil, err
		}

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
			// Publish queue gauges (depth, dead letters, backlog age).
			relay.SetInspector(outbox)
			app.wg.Add(1)
			go func() {
				defer app.wg.Done()
				app.runWorker(workerCtx, "outbox-relay", relay.Run)
			}()

			// Retention keeps the append-only tables bounded; leader-locked so only
			// one replica prunes at a time.
			retention := worker.NewRetention(retentionRepo, locker, log,
				cfg.Worker.RetentionInterval, cfg.Worker.RetentionBatch,
				cfg.Worker.OutboxRetention, cfg.Worker.AuditRetention)
			app.wg.Add(1)
			go func() {
				defer app.wg.Done()
				app.runWorker(workerCtx, "retention", retention.Run)
			}()

			// Referral commissions are paid into the referrer's wallet once the
			// cooling-off period (default 15 days) has elapsed. Leader-locked.
			settler := worker.NewCommissionSettler(commissionSvc, locker, log,
				cfg.Worker.CommissionSettleInterval, 100)
			app.wg.Add(1)
			go func() {
				defer app.wg.Done()
				app.runWorker(workerCtx, "commission-settler", settler.Run)
			}()
		}
	}

	log.Info("application initialised",
		"env", cfg.App.Env, "currency", cfg.App.Currency, "addr", addr,
		"storage", cfg.Storage.Driver, "paymentProvider", cfg.Payment.DefaultProvider,
		"workers", opts.RunWorkers && cfg.Worker.Enabled)
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
