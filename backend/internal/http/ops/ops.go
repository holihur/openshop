// Package ops builds the operations (admin) HTTP surface: the admin console SPA
// and the /api/v1/ops API. It is compiled into the openshop-ops binary only, so
// the admin surface can be deployed on an internal network independently of the
// public storefront.
package ops

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/docs"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	opsassets "github.com/holihur/openshop/internal/http/ops/assets"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// New builds the operations console engine.
func New(
	cfg *config.Config,
	tokens port.TokenIssuer,
	auth *service.AuthService,
	limiter port.RateLimiter,
	cache port.Cache,
	settings *service.SettingsService,
	metrics port.Metrics,
	tracer port.Tracer,
	shared *handler.Handler,
	analytics *service.AnalyticsService,
	roles port.RoleResolver,
) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	fh := &Handler{Handler: shared, Analytics: analytics}
	r := gin.New()
	r.RedirectTrailingSlash = false
	// Trust no proxy by default, so X-Forwarded-For cannot be spoofed to bypass
	// rate limits or forge audit entries. Set HTTP_TRUSTED_PROXIES behind an
	// ingress.
	_ = r.SetTrustedProxies(cfg.HTTP.TrustedProxies)
	// The console is same-origin (in dev the Vite server proxies to us), so no
	// CORS middleware is needed and none is enabled.
	r.Use(
		middleware.RequestID(),
		middleware.Tracing(tracer),
		middleware.Recovery(fh.Logger),
		middleware.Logger(fh.Logger),
		middleware.Metrics(metrics),
		middleware.Locale(),
		middleware.SecurityHeaders(cfg.App.IsProduction()),
		middleware.MaxBody(2<<20),
		middleware.RateLimit(limiter, handler.SettingLimit(settings, "security.rate_limit_rps")),
	)

	if fh.Metrics != nil {
		r.GET("/metrics", gin.WrapH(fh.Metrics))
	}
	r.GET("/healthz", fh.Healthz)
	r.GET("/readyz", fh.Readyz)
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerHTML))
	})

	api := r.Group("/api/v1")
	{
		api.GET("/version", fh.Version)
		api.GET("/openapi.yaml", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/yaml", docs.OpenAPISpec)
		})

		// The console signs in against the same auth service.
		// A tighter per-IP limit guards the console's sign-in too.
		api.POST("/auth/login", middleware.RateLimit(limiter, handler.SettingLimit(settings, "security.auth_rate_limit_rps")), fh.Login)
		api.POST("/auth/refresh", fh.Refresh)

		// A few storefront reads the console needs (category picker, currency
		// list). They are read-only and public on the storefront too.
		api.GET("/categories", fh.ListCategories)
		api.GET("/currencies", fh.ListCurrencies)

		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, fh.PATs, "ops", false))
		{
			authed.POST("/auth/logout", fh.Logout)
			authed.GET("/auth/me", fh.Me)
			authed.POST("/auth/password/change", fh.ChangePassword)
		}

		// Operations API: ops roles only, each route gated by a permission.
		// Idempotency makes mutating retries safe (a no-op without the header).
		admin := api.Group("/ops")
		admin.Use(
			middleware.Auth(tokens, auth, fh.PATs, "ops", false),
			middleware.RequireOps(roles),
			middleware.Idempotency(cache, 24*time.Hour),
		)
		{
			admin.POST("/categories", middleware.RequirePermission(roles, domain.PermCategoriesWrite), fh.CreateCategory)
			admin.PATCH("/categories/:id", middleware.RequirePermission(roles, domain.PermCategoriesWrite), fh.UpdateCategory)

			admin.GET("/products", middleware.RequirePermission(roles, domain.PermProductsRead), fh.ListProducts)
			admin.GET("/products/:id", middleware.RequirePermission(roles, domain.PermProductsRead), fh.GetProduct)
			admin.POST("/products", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.CreateProduct)
			admin.GET("/products/export.csv", middleware.RequirePermission(roles, domain.PermProductsRead), fh.ExportProducts)
			admin.POST("/products/import", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.ImportProducts)
			admin.PATCH("/products/:id", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.UpdateProduct)
			admin.GET("/products/:id/variants", middleware.RequirePermission(roles, domain.PermProductsRead), fh.ListVariants)
			admin.GET("/products/:id/faqs", middleware.RequirePermission(roles, domain.PermProductsRead), fh.ListProductFAQs)
			admin.PUT("/products/:id/faqs", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.ReplaceProductFAQs)
			admin.POST("/products/:id/variants", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.CreateVariant)
			admin.PATCH("/variants/:id", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.UpdateVariant)
			admin.POST("/uploads", middleware.RequirePermission(roles, domain.PermProductsWrite), fh.UploadImage)

			admin.GET("/orders", middleware.RequirePermission(roles, domain.PermOrdersRead), fh.ListOrders)
			admin.GET("/orders/:id", middleware.RequirePermission(roles, domain.PermOrdersRead), fh.GetOrder)
			admin.GET("/orders/:id/invoice", middleware.RequirePermission(roles, domain.PermOrdersRead), fh.DownloadInvoice)
			admin.POST("/orders/:id/refund", middleware.RequirePermission(roles, domain.PermRefundsWrite), fh.RefundOrder)
			admin.POST("/orders/:id/confirm-payment", middleware.RequirePermission(roles, domain.PermOrdersWrite), fh.ConfirmOrderPayment)
			admin.POST("/orders/:id/ship", middleware.RequirePermission(roles, domain.PermOrdersWrite), fh.ShipOrder)
			admin.POST("/orders/:id/complete", middleware.RequirePermission(roles, domain.PermOrdersWrite), fh.CompleteOrder)
			admin.GET("/returns", middleware.RequirePermission(roles, domain.PermReturnsRead), fh.ListReturns)
			admin.POST("/returns/:id/approve", middleware.RequirePermission(roles, domain.PermReturnsWrite), fh.ApproveReturn)
			admin.POST("/returns/:id/reject", middleware.RequirePermission(roles, domain.PermReturnsWrite), fh.RejectReturn)

			admin.GET("/tickets", middleware.RequirePermission(roles, domain.PermTicketsRead), fh.ListTickets)
			admin.GET("/tickets/:id", middleware.RequirePermission(roles, domain.PermTicketsRead), fh.GetTicket)
			admin.POST("/tickets/:id/messages", middleware.RequirePermission(roles, domain.PermTicketsWrite), fh.ReplyTicket)
			admin.PATCH("/tickets/:id", middleware.RequirePermission(roles, domain.PermTicketsWrite), fh.UpdateTicket)
			admin.POST("/tickets/:id/assign", middleware.RequirePermission(roles, domain.PermTicketsWrite), fh.AssignTicketToMe)

			admin.GET("/wallet/transactions", middleware.RequirePermission(roles, domain.PermLoyaltyRead), fh.ListWalletTransactions)
			admin.POST("/customers/:id/wallet/adjust", middleware.RequirePermission(roles, domain.PermLoyaltyWrite), fh.AdjustWallet)
			admin.POST("/customers/:id/points/adjust", middleware.RequirePermission(roles, domain.PermLoyaltyWrite), fh.AdjustPoints)
			admin.GET("/commissions", middleware.RequirePermission(roles, domain.PermLoyaltyRead), fh.ListCommissions)

			admin.GET("/withdrawals", middleware.RequirePermission(roles, domain.PermWithdrawalsRead), fh.ListWithdrawals)
			admin.POST("/withdrawals/:id/approve", middleware.RequirePermission(roles, domain.PermWithdrawalsWrite), fh.ApproveWithdrawal)
			admin.POST("/withdrawals/:id/reject", middleware.RequirePermission(roles, domain.PermWithdrawalsWrite), fh.RejectWithdrawal)
			admin.POST("/withdrawals/:id/pay", middleware.RequirePermission(roles, domain.PermWithdrawalsWrite), fh.PayWithdrawal)
			admin.POST("/notifications/broadcast", middleware.RequirePermission(roles, domain.PermNotificationsWrite), fh.BroadcastNotification)

			admin.GET("/token-scopes", fh.ListTokenScopes)
			admin.GET("/tokens", fh.ListOpsTokens)
			admin.POST("/tokens", fh.CreateOpsToken)
			admin.DELETE("/tokens/:id", fh.RevokeOpsToken)

			admin.GET("/customers", middleware.RequirePermission(roles, domain.PermCustomersRead), fh.ListCustomers)
			admin.GET("/customers/:id", middleware.RequirePermission(roles, domain.PermCustomersRead), fh.GetCustomer)
			admin.PATCH("/customers/:id", middleware.RequirePermission(roles, domain.PermCustomersWrite), fh.UpdateCustomer)

			admin.GET("/stats", middleware.RequirePermission(roles, domain.PermAnalyticsRead), fh.Dashboard)
			admin.GET("/stats/summary", middleware.RequirePermission(roles, domain.PermAnalyticsRead), fh.Summary)
			admin.GET("/inventory/low-stock", middleware.RequirePermission(roles, domain.PermAnalyticsRead), fh.LowStock)
			admin.GET("/reconciliation", middleware.RequirePermission(roles, domain.PermAuditRead), fh.ReconciliationReport)
			admin.GET("/audit-logs", middleware.RequirePermission(roles, domain.PermAuditRead), fh.ListAuditLogs)
			admin.GET("/audit-logs/verify", middleware.RequirePermission(roles, domain.PermAuditRead), fh.VerifyAuditChain)

			// The event queue: visibility and replay for dead-lettered events.
			admin.GET("/outbox", middleware.RequirePermission(roles, domain.PermAuditRead), fh.ListOutbox)
			admin.GET("/outbox/stats", middleware.RequirePermission(roles, domain.PermAuditRead), fh.OutboxStats)
			admin.POST("/outbox/:id/replay", middleware.RequirePermission(roles, domain.PermOutboxWrite), fh.ReplayOutbox)

			admin.GET("/staff", middleware.RequirePermission(roles, domain.PermStaffRead), fh.ListStaff)
			admin.POST("/staff", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.CreateStaff)
			admin.PATCH("/staff/:id", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.UpdateStaff)
			admin.POST("/staff/:id/password", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.ResetStaffPassword)
			admin.GET("/roles", middleware.RequirePermission(roles, domain.PermStaffRead), fh.ListRoles)
			admin.POST("/roles", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.CreateRole)
			admin.PATCH("/roles/:key", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.UpdateRole)
			admin.DELETE("/roles/:key", middleware.RequirePermission(roles, domain.PermStaffWrite), fh.DeleteRole)
			admin.GET("/settings", middleware.RequirePermission(roles, domain.PermSettingsRead), fh.ListSettings)
			admin.GET("/payment-gateways", middleware.RequirePermission(roles, domain.PermSettingsRead), fh.PaymentGateways)
			admin.PUT("/settings", middleware.RequirePermission(roles, domain.PermSettingsWrite), fh.UpdateSettings)

			admin.GET("/shipping-methods", middleware.RequirePermission(roles, domain.PermShippingRead), fh.AdminListShippingMethods)
			admin.POST("/shipping-methods", middleware.RequirePermission(roles, domain.PermShippingWrite), fh.CreateShippingMethod)
			admin.PATCH("/shipping-methods/:id", middleware.RequirePermission(roles, domain.PermShippingWrite), fh.UpdateShippingMethod)
			admin.GET("/shipping-zones", middleware.RequirePermission(roles, domain.PermShippingRead), fh.ListShippingZones)
			admin.POST("/shipping-zones", middleware.RequirePermission(roles, domain.PermShippingWrite), fh.CreateShippingZone)
			admin.PATCH("/shipping-zones/:id", middleware.RequirePermission(roles, domain.PermShippingWrite), fh.UpdateShippingZone)
			admin.PUT("/shipping-zones/:zoneId/rates/:methodId", middleware.RequirePermission(roles, domain.PermShippingWrite), fh.SetShippingRate)

			admin.PUT("/currencies/:code", middleware.RequirePermission(roles, domain.PermCurrencyWrite), fh.SetCurrencyRate)

			admin.GET("/coupons", middleware.RequirePermission(roles, domain.PermCouponsRead), fh.ListCoupons)
			admin.GET("/coupons/:id/redemptions", middleware.RequirePermission(roles, domain.PermCouponsRead), fh.ListCouponRedemptions)
			admin.POST("/coupons", middleware.RequirePermission(roles, domain.PermCouponsWrite), fh.CreateCoupon)
			admin.PATCH("/coupons/:id", middleware.RequirePermission(roles, domain.PermCouponsWrite), fh.UpdateCoupon)

			admin.GET("/reviews", middleware.RequirePermission(roles, domain.PermReviewsRead), fh.ListAllReviews)
			admin.DELETE("/reviews/:id", middleware.RequirePermission(roles, domain.PermReviewsWrite), fh.DeleteReview)
		}
	}

	// The admin console is served at the root of this (internal) listener.
	spa := opsassets.Handler()
	fh.Logger.Info("ops surface ready", "spa_embedded", opsassets.Built())
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "route not found"}})
			return
		}
		c.Status(http.StatusOK)
		spa.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
