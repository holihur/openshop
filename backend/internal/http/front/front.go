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
	frontassets "github.com/holihur/openshop/internal/http/front/assets"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// New builds the storefront engine.
func New(
	cfg *config.Config,
	tokens port.TokenIssuer,
	auth *service.AuthService,
	cache port.Cache,
	settings *service.SettingsService,
	limiter port.RateLimiter,
	metrics port.Metrics,
	tracer port.Tracer,
	shared *handler.Handler,
	cart *service.CartService,
	addresses *service.AddressService,
	wishlist *service.WishlistService,
	account *service.AccountService,
) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	fh := &Handler{Handler: shared, Cart: cart, Addresses: addresses, Wishlist: wishlist, Account: account}
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
		middleware.Recovery(fh.Logger),
		middleware.Logger(fh.Logger),
		middleware.Metrics(metrics),
		middleware.Locale(),
		middleware.CORS(cfg.HTTP.CORSOrigins),
		middleware.RateLimit(limiter, handler.SettingLimit(settings, "security.rate_limit_rps")),
	)

	if fh.Metrics != nil {
		r.GET("/metrics", gin.WrapH(fh.Metrics))
	}

	// Serve locally stored uploads. With the S3 driver the public URL points at
	// the bucket instead. The local handler resizes raster images on demand
	// (?w=NNN) for responsive srcset.
	if cfg.Storage.Driver == "" || cfg.Storage.Driver == "local" {
		uploads := gin.WrapH(mediaHandler(cfg.Storage.LocalDir))
		r.GET("/uploads/*filepath", uploads)
		r.HEAD("/uploads/*filepath", uploads)
	}

	r.GET("/healthz", fh.Healthz)
	r.GET("/readyz", fh.Readyz)
	r.GET("/robots.txt", fh.RobotsTxt)
	r.GET("/sitemap.xml", fh.Sitemap)
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(docs.SwaggerHTML))
	})

	api := r.Group("/api/v1")
	{
		api.GET("/version", fh.Version)
		api.GET("/openapi.yaml", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/yaml", docs.OpenAPISpec)
		})

		// Public catalog.
		api.GET("/categories", fh.ListCategories)
		api.GET("/site", fh.GetSite)
		api.GET("/products", fh.ListProducts)
		api.GET("/products/:id", fh.GetProduct)
		api.GET("/products/:id/reviews", fh.ListReviews)
		api.GET("/shipping-methods", fh.ListShippingMethods)
		api.GET("/payment-methods", fh.ListPaymentMethods)
		api.GET("/currencies", fh.ListCurrencies)

		// Auth.
		api.POST("/auth/register", fh.Register)
		api.POST("/auth/login", fh.Login)
		api.POST("/auth/refresh", fh.Refresh)
		api.POST("/auth/password/forgot", fh.ForgotPassword)
		api.POST("/auth/password/reset", fh.ResetPassword)
		api.POST("/auth/email/verify", fh.VerifyEmail)
		api.POST("/auth/email/resend", fh.ResendVerification)

		// OIDC single sign-on (authorization-code flow).
		api.GET("/auth/oidc/start", fh.OIDCStart)
		api.GET("/auth/oidc/callback", fh.OIDCCallback)

		// Provider callbacks are public and authenticated by signature.
		api.POST("/webhooks/payments/:provider", fh.PaymentWebhook)

		// Authenticated customer area.
		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, false), middleware.RateLimitUser(limiter, handler.SettingLimit(settings, "security.rate_limit_user_rps")))
		{
			authed.POST("/auth/logout", fh.Logout)
			authed.GET("/auth/me", fh.Me)
			authed.GET("/auth/me/export", fh.ExportAccount)
			authed.DELETE("/auth/me", fh.DeleteAccount)
			authed.POST("/auth/password/change", fh.ChangePassword)

			authed.GET("/wishlist", fh.ListWishlist)
			authed.POST("/wishlist", fh.AddWishlist)
			authed.DELETE("/wishlist/:productId", fh.RemoveWishlist)

			authed.GET("/orders", fh.ListOrders)
			authed.GET("/orders/:id", fh.GetOrder)
			authed.GET("/orders/:id/invoice", fh.DownloadInvoice)
			authed.POST("/orders/:id/cancel", fh.CancelOrder)
			authed.POST("/orders/:id/complete", fh.ConfirmReceipt)
			authed.POST("/orders/:id/returns", fh.RequestReturn)
			authed.GET("/orders/:id/returns", fh.ListOrderReturns)

			authed.POST("/payments", middleware.Idempotency(cache, 24*time.Hour), fh.CreatePayment)
			authed.POST("/payments/simulate", fh.SimulatePayment)

			authed.POST("/coupons/preview", fh.PreviewCoupon)
			authed.GET("/addresses", fh.ListAddresses)
			authed.POST("/addresses", fh.CreateAddress)
			authed.PATCH("/addresses/:id", fh.UpdateAddress)
			authed.DELETE("/addresses/:id", fh.DeleteAddress)
			authed.POST("/addresses/:id/default", fh.SetDefaultAddress)
			authed.POST("/products/:id/reviews", fh.AddReview)
			authed.PATCH("/reviews/:id", fh.UpdateReview)
			authed.DELETE("/reviews/:id", fh.DeleteReview)
		}

		// Cart & checkout: signed-in users or guests (X-Guest-Id header).
		shop := api.Group("")
		shop.Use(middleware.Auth(tokens, auth, true), middleware.ResolveSubject(), middleware.RequireSubject())
		{
			shop.GET("/cart", fh.GetCart)
			shop.POST("/cart/items", fh.AddCartItem)
			shop.PATCH("/cart/items/:productId", fh.UpdateCartItem)
			shop.DELETE("/cart/items/:productId", fh.RemoveCartItem)
			shop.DELETE("/cart", fh.ClearCart)
			shop.POST("/orders", middleware.Idempotency(cache, 24*time.Hour), fh.Checkout)
		}

		// Guest orders are authorised by an unguessable access token.
		api.GET("/guest/orders/:token", fh.GuestOrder)
		api.POST("/guest/orders/:token/pay", fh.GuestPay)
		api.POST("/guest/orders/:token/cancel", fh.GuestCancel)
		api.POST("/guest/orders/:token/complete", fh.GuestComplete)
		api.POST("/guest/payments/simulate", fh.GuestSimulate)
	}

	// Everything that is not an API, docs, metrics or upload route is served by
	// the embedded storefront SPA, falling back to index.html for client routes.
	spa := frontassets.Handler()
	fh.Logger.Info("storefront surface ready", "spa_embedded", frontassets.Built())
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
