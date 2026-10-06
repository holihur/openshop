// Package front builds the public storefront HTTP surface: the customer API,
// the storefront SPA, health/metrics/docs and the SEO endpoints. It is compiled
// into the openshop-server binary only; the operations API lives in the
// separate internal/http/ops package and openshop-ops binary.
package front

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/http/docs"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/web"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// New builds the storefront engine.
func New(
	cfg *config.Config,
	tokens port.TokenIssuer,
	auth *service.AuthService,
	cache port.Cache,
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
	// ClientIP (used by the per-IP rate limiter and audit) honours
	// X-Forwarded-For only from configured proxies.
	if len(cfg.HTTP.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.HTTP.TrustedProxies)
	}
	r.Use(
		middleware.RequestID(),
		middleware.Tracing(tracer),
		middleware.Recovery(h.Logger),
		middleware.Logger(h.Logger),
		middleware.Metrics(metrics),
		middleware.CORS(cfg.HTTP.CORSOrigins),
		middleware.RateLimit(limiter, cfg.HTTP.RateLimitRPS),
	)

	if h.Metrics != nil {
		r.GET("/metrics", gin.WrapH(h.Metrics))
	}

	// Serve locally stored uploads in development. With the S3 driver the
	// public URL points at the bucket instead.
	if cfg.Storage.Driver == "" || cfg.Storage.Driver == "local" {
		r.Static("/uploads", cfg.Storage.LocalDir)
	}

	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)
	r.GET("/robots.txt", h.RobotsTxt)
	r.GET("/sitemap.xml", h.Sitemap)
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerHTML))
	})

	api := r.Group("/api/v1")
	{
		api.GET("/version", h.Version)
		api.GET("/openapi.yaml", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/yaml", docs.OpenAPISpec)
		})

		// Public catalog.
		api.GET("/categories", h.ListCategories)
		api.GET("/products", h.ListProducts)
		api.GET("/products/:id", h.GetProduct)
		api.GET("/products/:id/reviews", h.ListReviews)
		api.GET("/shipping-methods", h.ListShippingMethods)
		api.GET("/currencies", h.ListCurrencies)

		// Auth.
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)
		api.POST("/auth/password/forgot", h.ForgotPassword)
		api.POST("/auth/password/reset", h.ResetPassword)
		api.POST("/auth/email/verify", h.VerifyEmail)
		api.POST("/auth/email/resend", h.ResendVerification)

		// Provider callbacks are public and authenticated by signature.
		api.POST("/webhooks/payments/:provider", h.PaymentWebhook)

		// Authenticated customer area.
		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, false), middleware.RateLimitUser(limiter, cfg.HTTP.RateLimitUserRPS))
		{
			authed.POST("/auth/logout", h.Logout)
			authed.GET("/auth/me", h.Me)
			authed.GET("/auth/me/export", h.ExportAccount)
			authed.DELETE("/auth/me", h.DeleteAccount)
			authed.POST("/auth/password/change", h.ChangePassword)

			authed.GET("/wishlist", h.ListWishlist)
			authed.POST("/wishlist", h.AddWishlist)
			authed.DELETE("/wishlist/:productId", h.RemoveWishlist)

			authed.GET("/orders", h.ListOrders)
			authed.GET("/orders/:id", h.GetOrder)
			authed.GET("/orders/:id/invoice", h.DownloadInvoice)
			authed.POST("/orders/:id/cancel", h.CancelOrder)
			authed.POST("/orders/:id/complete", h.ConfirmReceipt)

			authed.POST("/payments", middleware.Idempotency(cache, 24*time.Hour), h.CreatePayment)
			authed.POST("/payments/simulate", h.SimulatePayment)

			authed.POST("/coupons/preview", h.PreviewCoupon)
			authed.GET("/addresses", h.ListAddresses)
			authed.POST("/addresses", h.CreateAddress)
			authed.PATCH("/addresses/:id", h.UpdateAddress)
			authed.DELETE("/addresses/:id", h.DeleteAddress)
			authed.POST("/addresses/:id/default", h.SetDefaultAddress)
			authed.POST("/products/:id/reviews", h.AddReview)
			authed.PATCH("/reviews/:id", h.UpdateReview)
			authed.DELETE("/reviews/:id", h.DeleteReview)
		}

		// Cart & checkout: signed-in users or guests (X-Guest-Id header).
		shop := api.Group("")
		shop.Use(middleware.Auth(tokens, auth, true), middleware.ResolveSubject(), middleware.RequireSubject())
		{
			shop.GET("/cart", h.GetCart)
			shop.POST("/cart/items", h.AddCartItem)
			shop.PATCH("/cart/items/:productId", h.UpdateCartItem)
			shop.DELETE("/cart/items/:productId", h.RemoveCartItem)
			shop.DELETE("/cart", h.ClearCart)
			shop.POST("/orders", middleware.Idempotency(cache, 24*time.Hour), h.Checkout)
		}

		// Guest orders are authorised by an unguessable access token.
		api.GET("/guest/orders/:token", h.GuestOrder)
		api.POST("/guest/orders/:token/pay", h.GuestPay)
		api.POST("/guest/orders/:token/cancel", h.GuestCancel)
		api.POST("/guest/orders/:token/complete", h.GuestComplete)
		api.POST("/guest/payments/simulate", h.GuestSimulate)
	}

	// Everything that is not an API, docs, metrics or upload route is served by
	// the embedded storefront SPA, falling back to index.html for client routes.
	spa := web.Front()
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "route not found"}})
			return
		}
		// Gin sets a 404 before invoking NoRoute; reset it so the SPA handler
		// can serve its content with a 200.
		c.Status(http.StatusOK)
		spa.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
