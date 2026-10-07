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
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByPhone(ctx context.Context, phone string) (*domain.User, error)
}

// CategoryRepository persists categories.
type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	Update(ctx context.Context, c *domain.Category) error
	List(ctx context.Context) ([]domain.Category, error)
	FindByID(ctx context.Context, id string) (*domain.Category, error)
}

// ProductRepository persists products and owns atomic stock mutations.
type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	Update(ctx context.Context, p *domain.Product) error
	FindByID(ctx context.Context, id string) (*domain.Product, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Product, error)
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
	FindByAccessToken(ctx context.Context, token string) (*domain.Order, error)
	// FindExpiredPending returns pending orders past their expiry, used by the
	// auto-cancel worker. Scoped by limit so each run stays bounded.
	FindExpiredPending(ctx context.Context, now time.Time, limit int) ([]domain.Order, error)
	List(ctx context.Context, f domain.OrderFilter) (domain.Page[domain.Order], error)
	// AnonymizeByUser detaches a user's orders from their identity (GDPR erasure)
	// while keeping the financial record.
	AnonymizeByUser(ctx context.Context, userID string) error
	// HasPurchasedProduct reports whether a user has a paid order containing the
	// product, used to mark reviews as verified purchases.
	HasPurchasedProduct(ctx context.Context, userID, productID string) (bool, error)
}

// PaymentRepository persists payment attempts.
type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	Update(ctx context.Context, p *domain.Payment) error
	FindByID(ctx context.Context, id string) (*domain.Payment, error)
	FindByProviderRef(ctx context.Context, provider, ref string) (*domain.Payment, error)
}

// RefundRepository persists partial refunds against an order.
type RefundRepository interface {
	Create(ctx context.Context, r *domain.Refund) error
	ListByOrder(ctx context.Context, orderID string) ([]domain.Refund, error)
}

// ReturnRepository persists return requests (RMA).
type ReturnRepository interface {
	Create(ctx context.Context, r *domain.ReturnRequest) error
	FindByID(ctx context.Context, id string) (*domain.ReturnRequest, error)
	ListByOrder(ctx context.Context, orderID string) ([]domain.ReturnRequest, error)
	List(ctx context.Context, f domain.ReturnFilter) (domain.Page[domain.ReturnRequest], error)
	Update(ctx context.Context, r *domain.ReturnRequest) error
}

// RetentionRepository prunes append-only tables so they do not grow without
// limit. Deletes are batched to keep transactions short.
type RetentionRepository interface {
	DeletePublishedOutboxBefore(ctx context.Context, before time.Time, limit int) (int64, error)
	DeleteAuditBefore(ctx context.Context, before time.Time, limit int) (int64, error)
}

// CouponRepository persists coupons and their redemptions. IncrementUsage is
// the distributed-safe guard that enforces global usage limits.
type CouponRepository interface {
	Create(ctx context.Context, c *domain.Coupon) error
	Update(ctx context.Context, c *domain.Coupon) error
	FindByID(ctx context.Context, id string) (*domain.Coupon, error)
	FindByCode(ctx context.Context, code string) (*domain.Coupon, error)
	List(ctx context.Context, f domain.CouponFilter) (domain.Page[domain.Coupon], error)
	// IncrementUsage atomically consumes one redemption. It returns
	// ErrCouponExhausted when the global limit has been reached, which makes it
	// correct under concurrency across any number of replicas.
	IncrementUsage(ctx context.Context, id string) error
	CountRedemptions(ctx context.Context, couponID, userID string) (int64, error)
	CreateRedemption(ctx context.Context, r *domain.CouponRedemption) error
}

// ReviewRepository persists product reviews and their aggregate.
type ReviewRepository interface {
	Create(ctx context.Context, r *domain.Review) error
	Update(ctx context.Context, r *domain.Review) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*domain.Review, error)
	FindByUserAndProduct(ctx context.Context, userID, productID string) (*domain.Review, error)
	ListByProduct(ctx context.Context, f domain.ReviewFilter) (domain.Page[domain.Review], error)
	// List returns reviews across products for moderation; an empty ProductID
	// means all products.
	List(ctx context.Context, f domain.ReviewFilter) (domain.Page[domain.Review], error)
	ListByUser(ctx context.Context, userID string) ([]domain.Review, error)
	DeleteByUser(ctx context.Context, userID string) error
	Summary(ctx context.Context, productID string) (domain.ReviewSummary, error)
}

// VariantRepository persists product variants and owns variant-level stock
// mutations, which are the authoritative inventory for multi-variant products.
type VariantRepository interface {
	Create(ctx context.Context, v *domain.Variant) error
	Update(ctx context.Context, v *domain.Variant) error
	FindByID(ctx context.Context, id string) (*domain.Variant, error)
	ListByProduct(ctx context.Context, productID string) ([]domain.Variant, error)
	// DecreaseStock atomically decrements variant stock, returning
	// ErrInsufficientStock when the guard fails.
	DecreaseStock(ctx context.Context, variantID string, quantity int) error
	IncreaseStock(ctx context.Context, variantID string, quantity int) error
}

// AddressRepository persists a user's shipping address book.
type AddressRepository interface {
	Create(ctx context.Context, a *domain.Address) error
	Update(ctx context.Context, a *domain.Address) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*domain.Address, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Address, error)
	// ClearDefault unsets the default flag for every address of a user, so a new
	// default can be set atomically by the caller.
	ClearDefault(ctx context.Context, userID string) error
	DeleteByUser(ctx context.Context, userID string) error
}

// AnalyticsRepository provides aggregate reads for the merchant dashboard.
type AnalyticsRepository interface {
	Dashboard(ctx context.Context) (domain.Dashboard, error)
	// LowStock lists products and variants at or below the threshold.
	LowStock(ctx context.Context, threshold int) ([]domain.LowStockItem, error)
}

// ShippingMethodRepository persists selectable shipping options.
type ShippingMethodRepository interface {
	Create(ctx context.Context, m *domain.ShippingMethod) error
	Update(ctx context.Context, m *domain.ShippingMethod) error
	FindByID(ctx context.Context, id string) (*domain.ShippingMethod, error)
	List(ctx context.Context, activeOnly bool) ([]domain.ShippingMethod, error)
	// Default returns the cheapest active method by sort order, used when the
	// customer does not choose one.
	Default(ctx context.Context) (*domain.ShippingMethod, error)
}

// ShippingZoneRepository persists zones and their per-method rates.
type ShippingZoneRepository interface {
	Create(ctx context.Context, z *domain.ShippingZone) error
	Update(ctx context.Context, z *domain.ShippingZone) error
	List(ctx context.Context, activeOnly bool) ([]domain.ShippingZone, error)
	FindByID(ctx context.Context, id string) (*domain.ShippingZone, error)
	// FindByProvince returns the first active zone covering a province.
	FindByProvince(ctx context.Context, province string) (*domain.ShippingZone, error)
	UpsertRate(ctx context.Context, r *domain.ShippingRate) error
	FindRate(ctx context.Context, zoneID, methodID string) (*domain.ShippingRate, error)
}

// AuditRepository persists the append-only audit trail.
type AuditRepository interface {
	Create(ctx context.Context, entry *domain.AuditLog) error
	List(ctx context.Context, f domain.AuditFilter) (domain.Page[domain.AuditLog], error)
}

// WishlistRepository persists a user's saved products.
type WishlistRepository interface {
	Add(ctx context.Context, userID, productID string) error
	Remove(ctx context.Context, userID, productID string) error
	ListByUser(ctx context.Context, userID string) ([]domain.Product, error)
	DeleteByUser(ctx context.Context, userID string) error
}

// CurrencyRepository persists exchange rates from the base currency.
type CurrencyRepository interface {
	Upsert(ctx context.Context, currency string, rateMicro int64) error
	List(ctx context.Context) ([]domain.ExchangeRate, error)
	Find(ctx context.Context, currency string) (*domain.ExchangeRate, error)
}

// ExchangeRates resolves a micro rate between two currencies. It is used by
// checkout to settle orders in the customer's chosen currency.
type ExchangeRates interface {
	Rate(ctx context.Context, from, to string) (int64, error)
}
