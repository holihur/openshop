// Package handler contains the Gin HTTP handlers. Handlers translate between
// HTTP and the service layer only; they contain no business rules.
package handler

import (
	"context"

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
	Auth     *service.AuthService
	Catalog  *service.CatalogService
	Cart     *service.CartService
	Orders   *service.OrderService
	Payments *service.PaymentService
	Storage  port.ObjectStorage
	IDs      port.IDGenerator
	Logger   port.Logger
	Checks   []ReadinessCheck
}
