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
	"github.com/holihur/openshop/internal/http/web"
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
	h *handler.Handler,
) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

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
		middleware.Recovery(h.Logger),
		middleware.Logger(h.Logger),
		middleware.Metrics(metrics),
		middleware.RateLimit(limiter, cfg.HTTP.RateLimitRPS),
	)

	if h.Metrics != nil {
		r.GET("/metrics", gin.WrapH(h.Metrics))
	}
	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerHTML))
	})

	api := r.Group("/api/v1")
	{
		api.GET("/version", h.Version)
		api.GET("/openapi.yaml", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/yaml", docs.OpenAPISpec)
		})

		// The console signs in against the same auth service.
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)

		// A few storefront reads the console needs (category picker, currency
		// list). They are read-only and public on the storefront too.
		api.GET("/categories", h.ListCategories)
		api.GET("/currencies", h.ListCurrencies)

		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, false))
		{
			authed.POST("/auth/logout", h.Logout)
			authed.GET("/auth/me", h.Me)
			authed.POST("/auth/password/change", h.ChangePassword)
		}

		// Operations API: admin-only.
		admin := api.Group("/ops")
		admin.Use(middleware.Auth(tokens, auth, false), middleware.RequireAdmin())
		{
			admin.POST("/categories", h.CreateCategory)

			admin.GET("/products", h.ListProducts)
			admin.POST("/products", h.CreateProduct)
			admin.PATCH("/products/:id", h.UpdateProduct)
			admin.GET("/products/:id/variants", h.ListVariants)
			admin.POST("/products/:id/variants", h.CreateVariant)
			admin.PATCH("/variants/:id", h.UpdateVariant)
			admin.POST("/uploads", h.UploadImage)

			admin.GET("/orders", h.ListOrders)
			admin.GET("/orders/:id/invoice", h.DownloadInvoice)
			admin.POST("/orders/:id/refund", h.RefundOrder)
			admin.POST("/orders/:id/ship", h.ShipOrder)
			admin.POST("/orders/:id/complete", h.CompleteOrder)

			admin.GET("/stats", h.Dashboard)
			admin.GET("/inventory/low-stock", h.LowStock)
			admin.GET("/audit-logs", h.ListAuditLogs)

			admin.GET("/shipping-methods", h.AdminListShippingMethods)
			admin.POST("/shipping-methods", h.CreateShippingMethod)
			admin.PATCH("/shipping-methods/:id", h.UpdateShippingMethod)
			admin.GET("/shipping-zones", h.ListShippingZones)
			admin.POST("/shipping-zones", h.CreateShippingZone)
			admin.PATCH("/shipping-zones/:id", h.UpdateShippingZone)
			admin.PUT("/shipping-zones/:zoneId/rates/:methodId", h.SetShippingRate)

			admin.PUT("/currencies/:code", h.SetCurrencyRate)

			admin.GET("/coupons", h.ListCoupons)
			admin.POST("/coupons", h.CreateCoupon)
			admin.PATCH("/coupons/:id", h.UpdateCoupon)

			admin.GET("/reviews", h.ListAllReviews)
		}
	}

	// The admin console is served at the root of this (internal) listener.
	spa := web.Ops()
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
