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

// Handler bundles the shared services and infrastructure the HTTP layer needs.
// Surface-specific services (cart/wishlist/address/account for the storefront,
// analytics for ops) live on the surface handler that embeds this one.
type Handler struct {
	Auth      *service.AuthService
	Catalog   *service.CatalogService
	Orders    *service.OrderService
	Payments  *service.PaymentService
	Coupons   *service.CouponService
	Reviews   *service.ReviewService
	Shipping  *service.ShippingService
	Audit     *service.AuditService
	Currency  *service.CurrencyService
	Returns   *service.ReturnService
	Settings  *service.SettingsService
	Customers *service.CustomerService
	OIDC      *service.OIDCService
	Storage   port.ObjectStorage
	Cache     port.Cache
	IDs       port.IDGenerator
	Logger    port.Logger
	Metrics   http.Handler
	Checks    []ReadinessCheck
}
