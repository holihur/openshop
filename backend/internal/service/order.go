package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// OrderService owns checkout and the order lifecycle. It demonstrates the two
// primitives required for horizontal scaling: a distributed lock for
// user-level mutual exclusion and a database transaction for atomic stock
// decrement plus order insert.
type OrderService struct {
	orders   port.OrderRepository
	products port.ProductRepository
	carts    port.CartRepository
	locker   port.Locker
	tx       port.TxManager
	bus      port.EventBus
	ids      port.IDGenerator
	clock    port.Clock
	logger   port.Logger
	ttl      time.Duration
	currency string
}

func NewOrderService(
	orders port.OrderRepository,
	products port.ProductRepository,
	carts port.CartRepository,
	locker port.Locker,
	tx port.TxManager,
	bus port.EventBus,
	ids port.IDGenerator,
	clock port.Clock,
	logger port.Logger,
	ttl time.Duration,
	currency string,
) *OrderService {
	return &OrderService{
		orders: orders, products: products, carts: carts, locker: locker, tx: tx, bus: bus,
		ids: ids, clock: clock, logger: logger, ttl: ttl, currency: currency,
	}
}

// Checkout converts the user's cart into a pending order, reserving stock. The
// per-user lock prevents duplicate submissions from retries or double clicks;
// the transaction guarantees stock and order move together.
func (s *OrderService) Checkout(ctx context.Context, userID string) (*domain.Order, error) {
	lock, err := s.locker.Acquire(ctx, "lock:checkout:"+userID, 15*time.Second, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release(ctx) }()

	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}

	now := s.clock.Now()
	order := &domain.Order{
		ID:        s.ids.NewID(),
		OrderNo:   orderNo(now, s.ids.NewID()),
		UserID:    userID,
		Status:    domain.OrderPendingPayment,
		Currency:  s.currency,
		ExpiresAt: now.Add(s.ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		var total int64
		items := make([]domain.OrderItem, 0, len(cart.Items))
		for _, ci := range cart.Items {
			// Re-read the product so pricing and stock are authoritative at the
			// moment of purchase, not when the item was added to the cart.
			p, err := s.products.FindByID(txCtx, ci.ProductID)
			if err != nil {
				return fmt.Errorf("load product %s: %w", ci.ProductID, err)
			}
			if p.Status != domain.ProductPublished {
				return fmt.Errorf("%w: %s is no longer available", domain.ErrInvalidArgument, p.Title)
			}
			if err := s.products.DecreaseStock(txCtx, p.ID, ci.Quantity); err != nil {
				return err
			}
			subtotal := p.PriceCents * int64(ci.Quantity)
			total += subtotal
			items = append(items, domain.OrderItem{
				ID: s.ids.NewID(), OrderID: order.ID, ProductID: p.ID, Title: p.Title,
				PriceCents: p.PriceCents, Quantity: ci.Quantity, Subtotal: subtotal,
			})
		}
		if total <= 0 {
			return fmt.Errorf("%w: order total must be positive", domain.ErrInvalidArgument)
		}
		order.TotalCents = total
		order.Items = items
		return s.orders.Create(txCtx, order)
	})
	if err != nil {
		return nil, err
	}

	// The cart is cleared only after the transaction commits. Redis is shared,
	// so the change is visible to the user's next request on any instance.
	if err := s.carts.Delete(ctx, userID); err != nil {
		s.logger.Warn("failed to clear cart after checkout", "userId", userID, "error", err)
	}

	s.publish(ctx, SubjectOrderCreated, order)
	return order, nil
}

func (s *OrderService) Get(ctx context.Context, userID, orderID string, isAdmin bool) (*domain.Order, error) {
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && o.UserID != userID {
		// Do not leak existence of another user's order.
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *OrderService) List(ctx context.Context, f domain.OrderFilter) (domain.Page[domain.Order], error) {
	f.Page, f.PageSize = clampPage(f.Page, f.PageSize, 10)
	return s.orders.List(ctx, f)
}

// Cancel transitions a pending order to cancelled and returns reserved stock.
// It is safe to call concurrently: a distributed lock serialises the
// transition and a status check prevents double refunds of stock.
func (s *OrderService) Cancel(ctx context.Context, userID, orderID string, isAdmin bool) (*domain.Order, error) {
	lock, err := s.locker.Acquire(ctx, "lock:order:"+orderID, 10*time.Second, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release(ctx) }()

	var out *domain.Order
	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		o, err := s.orders.FindByID(txCtx, orderID)
		if err != nil {
			return err
		}
		if !isAdmin && o.UserID != userID {
			return domain.ErrNotFound
		}
		if !o.Cancelable() {
			return fmt.Errorf("%w: order cannot be cancelled", domain.ErrConflict)
		}
		if err := s.releaseStock(txCtx, o); err != nil {
			return err
		}
		o.Status = domain.OrderCancelled
		o.UpdatedAt = s.clock.Now()
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publish(ctx, SubjectOrderCancelled, out)
	return out, nil
}

// MarkPaid is invoked by the payment service after a verified provider
// callback. It is idempotent, so duplicate webhooks are harmless.
func (s *OrderService) MarkPaid(ctx context.Context, orderID, paymentID string) (*domain.Order, error) {
	lock, err := s.locker.Acquire(ctx, "lock:order:"+orderID, 10*time.Second, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release(ctx) }()

	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status == domain.OrderPaid {
		return o, nil // already processed
	}
	if !o.Payable() {
		return nil, domain.ErrOrderNotPayable
	}
	now := s.clock.Now()
	o.Status = domain.OrderPaid
	o.PaymentID = paymentID
	o.PaidAt = &now
	o.UpdatedAt = now
	if err := s.orders.Update(ctx, o); err != nil {
		return nil, err
	}
	s.publish(ctx, SubjectOrderPaid, o)
	return o, nil
}

// CancelExpired is called by the background sweeper. It releases stock for
// orders whose payment window elapsed.
func (s *OrderService) CancelExpired(ctx context.Context, orderID string) error {
	_, err := s.Cancel(ctx, "", orderID, true)
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict) {
		return nil
	}
	return err
}

// ListExpired exposes the repository query to the worker.
func (s *OrderService) ListExpired(ctx context.Context, now time.Time, limit int) ([]domain.Order, error) {
	return s.orders.FindExpiredPending(ctx, now, limit)
}

func (s *OrderService) releaseStock(ctx context.Context, o *domain.Order) error {
	for _, item := range o.Items {
		if err := s.products.IncreaseStock(ctx, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func (s *OrderService) publish(ctx context.Context, subject string, o *domain.Order) {
	items := make([]OrderEventItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderEventItem{
			ProductID: it.ProductID, Title: it.Title, Quantity: it.Quantity, PriceCents: it.PriceCents,
		})
	}
	evt := OrderEvent{
		OrderID: o.ID, OrderNo: o.OrderNo, UserID: o.UserID, Status: string(o.Status),
		TotalCents: o.TotalCents, Currency: o.Currency, PaymentID: o.PaymentID,
		OccurredAt: s.clock.Now().Format(time.RFC3339), Items: items,
	}
	// Publishing must never fail checkout; the event bus retries and consumers
	// are idempotent. We log and move on.
	if err := s.bus.Publish(ctx, port.Event{ID: s.ids.NewID(), Subject: subject, Payload: encodeEvent(evt)}); err != nil {
		s.logger.Error("failed to publish order event", "subject", subject, "orderId", o.ID, "error", err)
	}
}

// orderNo builds a human-sortable, collision-resistant order number.
func orderNo(now time.Time, id string) string {
	suffix := id
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return "OS" + now.Format("20060102150405") + suffix
}
