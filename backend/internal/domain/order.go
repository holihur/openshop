package domain

import "time"

type OrderStatus string

const (
	OrderPendingPayment OrderStatus = "pending_payment"
	OrderPaid           OrderStatus = "paid"
	OrderCancelled      OrderStatus = "cancelled"
	OrderShipped        OrderStatus = "shipped"
	OrderCompleted      OrderStatus = "completed"
	OrderRefunded       OrderStatus = "refunded"
)

type OrderItem struct {
	ID          string
	OrderID     string
	ProductID   string
	VariantID   string
	VariantName string
	SKU         string
	Title       string
	PriceCents  int64
	Quantity    int
	Subtotal    int64
}

type Order struct {
	ID       string
	OrderNo  string
	UserID   string
	Status   OrderStatus
	Currency string
	// SubtotalCents is the pre-discount total; TotalCents is what the customer
	// pays after DiscountCents is applied.
	SubtotalCents int64
	DiscountCents int64
	CouponID      string
	CouponCode    string
	TotalCents    int64
	Items         []OrderItem
	PaymentID     string
	// ShippingAddress is a snapshot taken at checkout (nil for digital orders).
	ShippingAddress *Address
	TrackingNo      string
	ShippedAt       *time.Time
	CompletedAt     *time.Time
	// ExpiresAt drives the "auto cancel unpaid order" worker. It is persisted so
	// any instance can reclaim the released inventory after a restart.
	ExpiresAt time.Time
	PaidAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (o *Order) Payable() bool {
	return o.Status == OrderPendingPayment
}

func (o *Order) Cancelable() bool {
	return o.Status == OrderPendingPayment
}

func (o *Order) Refundable() bool {
	switch o.Status {
	case OrderPaid, OrderShipped, OrderCompleted:
		return true
	default:
		return false
	}
}

func (o *Order) Shippable() bool { return o.Status == OrderPaid }

func (o *Order) Completetable() bool { return o.Status == OrderShipped }

func (o *Order) Expired(now time.Time) bool {
	return o.Status == OrderPendingPayment && !o.ExpiresAt.IsZero() && now.After(o.ExpiresAt)
}

// OrderFilter is a storage agnostic query for the order list endpoint.
type OrderFilter struct {
	UserID   string
	Status   *OrderStatus
	Page     int
	PageSize int
}
