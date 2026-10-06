package port

import (
	"context"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// CartRepository persists the active cart. It is intentionally a port so the
// storage backend (Redis today, DynamoDB tomorrow) never leaks into services.
type CartRepository interface {
	Get(ctx context.Context, userID string) (*domain.Cart, error)
	Save(ctx context.Context, cart *domain.Cart) error
	Delete(ctx context.Context, userID string) error
}

// UserRepository persists accounts. Implemented by the GORM adapter.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	Update(ctx context.Context, u *domain.User) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
}

// CategoryRepository persists categories.
type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	List(ctx context.Context) ([]domain.Category, error)
	FindByID(ctx context.Context, id string) (*domain.Category, error)
}

// ProductRepository persists products and owns atomic stock mutations.
type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	Update(ctx context.Context, p *domain.Product) error
	FindByID(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context, f domain.ProductFilter) (domain.Page[domain.Product], error)
	// DecreaseStock atomically decrements stock when enough is available. It
	// returns ErrInsufficientStock when the guard fails, which makes it safe for
	// concurrent, multi-instance checkouts.
	DecreaseStock(ctx context.Context, productID string, quantity int) error
	IncreaseStock(ctx context.Context, productID string, quantity int) error
}

// OrderRepository persists orders and their items.
type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	Update(ctx context.Context, o *domain.Order) error
	FindByID(ctx context.Context, id string) (*domain.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*domain.Order, error)
	// FindExpiredPending returns pending orders past their expiry, used by the
	// auto-cancel worker. Scoped by limit so each run stays bounded.
	FindExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error)
	List(ctx context.Context, f domain.OrderFilter) (domain.Page[domain.Order], error)
}

// PaymentRepository persists payment attempts.
type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	Update(ctx context.Context, p *domain.Payment) error
	FindByID(ctx context.Context, id string) (*domain.Payment, error)
	FindByProviderRef(ctx context.Context, provider, ref string) (*domain.Payment, error)
}
