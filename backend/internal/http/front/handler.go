package front

import (
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/service"
)

// Handler carries the storefront-specific handlers and services on top of the
// shared base. Shared handlers (login, catalog reads, order reads) are promoted
// from the embedded *handler.Handler.
type Handler struct {
	*handler.Handler

	Cart      *service.CartService
	Addresses *service.AddressService
	Wishlist  *service.WishlistService
	Account   *service.AccountService
}
