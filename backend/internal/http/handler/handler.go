// Package handler contains the Gin HTTP handlers. Handlers translate between
// HTTP and the service layer only; they contain no business rules.
package handler

import (
	"context"
	"net/http"

	"github.com/holihur/openshop/internal/port"
	"github.com/holihur/openshop/internal/service"
)

// ReadinessCheck is a named dependency probe surfaced by /readyz.
type ReadinessCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// Handler bundles the services and infrastructure the HTTP layer needs.
type Handler struct {
	Auth      *service.AuthService
	Catalog   *service.CatalogService
	Cart      *service.CartService
	Orders    *service.OrderService
	Payments  *service.PaymentService
	Coupons   *service.CouponService
	Reviews   *service.ReviewService
	Addresses *service.AddressService
	Analytics *service.AnalyticsService
	Shipping  *service.ShippingService
	Audit     *service.AuditService
	Wishlist  *service.WishlistService
	Currency  *service.CurrencyService
	Storage   port.ObjectStorage
	SiteURL   string
	IDs       port.IDGenerator
	Logger    port.Logger
	Metrics   http.Handler
	Checks    []ReadinessCheck
}
