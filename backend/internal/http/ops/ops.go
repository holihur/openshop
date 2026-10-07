// Package ops builds the operations (admin) HTTP surface: the admin console SPA
// and the /api/v1/ops API. It is compiled into the openshop-ops binary only, so
// the admin surface can be deployed on an internal network independently of the
// public storefront.
package ops

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/config"
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
	metrics port.Metrics,
	tracer port.Tracer,
	shared *handler.Handler,
	analytics *service.AnalyticsService,
) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	fh := &Handler{Handler: shared, Analytics: analytics}
	r := gin.New()
	r.RedirectTrailingSlash = false
	if len(cfg.HTTP.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.HTTP.TrustedProxies)
	}
	// The console is same-origin (in dev the Vite server proxies to us), so no
	// CORS middleware is needed and none is enabled.
	r.Use(
		middleware.RequestID(),
		middleware.Tracing(tracer),
		middleware.Recovery(fh.Logger),
		middleware.Logger(fh.Logger),
		middleware.Metrics(metrics),
		middleware.RateLimit(limiter, cfg.HTTP.RateLimitRPS),
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
		api.POST("/auth/login", fh.Login)
		api.POST("/auth/refresh", fh.Refresh)

		// A few storefront reads the console needs (category picker, currency
		// list). They are read-only and public on the storefront too.
		api.GET("/categories", fh.ListCategories)
		api.GET("/currencies", fh.ListCurrencies)

		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, false))
		{
			authed.POST("/auth/logout", fh.Logout)
			authed.GET("/auth/me", fh.Me)
			authed.POST("/auth/password/change", fh.ChangePassword)
		}

		// Operations API: admin-only.
		admin := api.Group("/ops")
		admin.Use(middleware.Auth(tokens, auth, false), middleware.RequireAdmin())
		{
			admin.POST("/categories", fh.CreateCategory)

			admin.GET("/products", fh.ListProducts)
			admin.POST("/products", fh.CreateProduct)
			admin.PATCH("/products/:id", fh.UpdateProduct)
			admin.GET("/products/:id/variants", fh.ListVariants)
			admin.POST("/products/:id/variants", fh.CreateVariant)
			admin.PATCH("/variants/:id", fh.UpdateVariant)
			admin.POST("/uploads", fh.UploadImage)

			admin.GET("/orders", fh.ListOrders)
			admin.GET("/orders/:id", fh.GetOrder)
			admin.GET("/orders/:id/invoice", fh.DownloadInvoice)
			admin.POST("/orders/:id/refund", fh.RefundOrder)
			admin.POST("/orders/:id/ship", fh.ShipOrder)
			admin.POST("/orders/:id/complete", fh.CompleteOrder)
			admin.GET("/returns", fh.ListReturns)
			admin.POST("/returns/:id/approve", fh.ApproveReturn)
			admin.POST("/returns/:id/reject", fh.RejectReturn)

			admin.GET("/stats", fh.Dashboard)
			admin.GET("/inventory/low-stock", fh.LowStock)
			admin.GET("/audit-logs", fh.ListAuditLogs)

			admin.GET("/shipping-methods", fh.AdminListShippingMethods)
			admin.POST("/shipping-methods", fh.CreateShippingMethod)
			admin.PATCH("/shipping-methods/:id", fh.UpdateShippingMethod)
			admin.GET("/shipping-zones", fh.ListShippingZones)
			admin.POST("/shipping-zones", fh.CreateShippingZone)
			admin.PATCH("/shipping-zones/:id", fh.UpdateShippingZone)
			admin.PUT("/shipping-zones/:zoneId/rates/:methodId", fh.SetShippingRate)

			admin.PUT("/currencies/:code", fh.SetCurrencyRate)

			admin.GET("/coupons", fh.ListCoupons)
			admin.POST("/coupons", fh.CreateCoupon)
			admin.PATCH("/coupons/:id", fh.UpdateCoupon)

			admin.GET("/reviews", fh.ListAllReviews)
			admin.DELETE("/reviews/:id", fh.DeleteReview)
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
