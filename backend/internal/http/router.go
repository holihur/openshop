// Package http wires the Gin engine: middleware chain and route table. It is
// the only package that imports Gin, keeping the framework at the edge.
package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/http/docs"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// NewRouter builds the HTTP engine. The same engine is created on every
// replica; there is no instance-local state in the routing layer.
func NewRouter(
	cfg *config.Config,
	tokens port.TokenIssuer,
	auth *service.AuthService,
	cache port.Cache,
	metrics port.Metrics,
	h *handler.Handler,
) *gin.Engine {
	if cfg.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(h.Logger),
		middleware.Logger(h.Logger),
		middleware.Metrics(metrics),
		middleware.CORS(cfg.HTTP.CORSOrigins),
		middleware.RateLimit(cache, cfg.HTTP.RateLimitRPS),
	)

	// Prometheus exposition endpoint.
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

	// API documentation: interactive UI and the raw OpenAPI document.
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

		// Auth.
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.Refresh)

		// Provider callbacks are public and authenticated by signature.
		api.POST("/webhooks/payments/:provider", h.PaymentWebhook)

		// Authenticated customer area.
		authed := api.Group("")
		authed.Use(middleware.Auth(tokens, auth, false))
		{
			authed.POST("/auth/logout", h.Logout)
			authed.GET("/auth/me", h.Me)

			authed.GET("/cart", h.GetCart)
			authed.POST("/cart/items", h.AddCartItem)
			authed.PATCH("/cart/items/:productId", h.UpdateCartItem)
			authed.DELETE("/cart/items/:productId", h.RemoveCartItem)
			authed.DELETE("/cart", h.ClearCart)

			authed.POST("/orders", middleware.Idempotency(cache, 24*time.Hour), h.Checkout)
			authed.GET("/orders", h.ListOrders)
			authed.GET("/orders/:id", h.GetOrder)
			authed.POST("/orders/:id/cancel", h.CancelOrder)

			authed.POST("/payments", middleware.Idempotency(cache, 24*time.Hour), h.CreatePayment)
			authed.POST("/payments/simulate", h.SimulatePayment)
		}

		// Admin area.
		admin := api.Group("/admin")
		admin.Use(middleware.Auth(tokens, auth, false), middleware.RequireAdmin())
		{
			admin.POST("/categories", h.CreateCategory)
			admin.GET("/products", h.ListProducts)
			admin.POST("/products", h.CreateProduct)
			admin.PATCH("/products/:id", h.UpdateProduct)
			admin.POST("/uploads", h.UploadImage)
			admin.GET("/orders", h.ListOrders)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "route not found"}})
	})

	return r
}
