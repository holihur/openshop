package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ProductCacheInvalidator evicts cached product views. It is implemented by
// CatalogService and injected so that checkout can reflect stock changes
// immediately, rather than waiting for the asynchronous event consumer.
type ProductCacheInvalidator interface {
	InvalidateProductCache(ctx context.Context, ids ...string)
}

// OrderService owns checkout and the order lifecycle. It demonstrates the two
// primitives required for horizontal scaling: a distributed lock for
// user-level mutual exclusion and a database transaction for atomic stock
// decrement plus order insert. Lifecycle events are written to the
// transactional outbox in the same transaction, so they are never lost.
type OrderService struct {
	orders       port.OrderRepository
	products     port.ProductRepository
	coupons      port.CouponRepository
	variants     port.VariantRepository
	carts        port.CartRepository
	locker       port.Locker
	tx           port.TxManager
	outbox       port.Outbox
	ids          port.IDGenerator
	clock        port.Clock
	logger       port.Logger
	productCache ProductCacheInvalidator
	metrics      port.Metrics
	tracer       port.Tracer
	ttl          time.Duration
	currency     string
}

func NewOrderService(
	orders port.OrderRepository,
	products port.ProductRepository,
	coupons port.CouponRepository,
	variants port.VariantRepository,
	carts port.CartRepository,
	locker port.Locker,
	tx port.TxManager,
	outbox port.Outbox,
	ids port.IDGenerator,
	clock port.Clock,
	logger port.Logger,
	productCache ProductCacheInvalidator,
	metrics port.Metrics,
	tracer port.Tracer,
	ttl time.Duration,
	currency string,
) *OrderService {
	if metrics == nil {
		metrics = port.NopMetrics{}
	}
	if tracer == nil {
		tracer = port.NoopTracer{}
	}
	return &OrderService{
		orders: orders, products: products, coupons: coupons, variants: variants, carts: carts, locker: locker, tx: tx,
		outbox: outbox, ids: ids, clock: clock, logger: logger, productCache: productCache,
		metrics: metrics, tracer: tracer, ttl: ttl, currency: currency,
	}
}

// Checkout converts the user's cart into a pending order, reserving stock. The
// per-user lock prevents duplicate submissions from retries or double clicks;
// the transaction guarantees stock and order move together.
func (s *OrderService) Checkout(ctx context.Context, userID, couponCode string) (out *domain.Order, err error) {
	ctx, span := s.tracer.Start(ctx, "order.checkout",
		port.Attribute{Key: "user.id", Value: userID},
		port.Attribute{Key: "coupon", Value: couponCode},
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

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

			price := p.PriceCents
			item := domain.OrderItem{
				ID: s.ids.NewID(), OrderID: order.ID, ProductID: p.ID, Title: p.Title,
				Quantity: ci.Quantity,
			}
			if ci.VariantID != "" {
				v, err := s.variants.FindByID(txCtx, ci.VariantID)
				if err != nil {
					return fmt.Errorf("load variant %s: %w", ci.VariantID, err)
				}
				if v.ProductID != p.ID || !v.Active {
					return fmt.Errorf("%w: variant is no longer available", domain.ErrInvalidArgument)
				}
				if err := s.variants.DecreaseStock(txCtx, v.ID, ci.Quantity); err != nil {
					return err
				}
				price = v.EffectivePrice(p.PriceCents)
				item.VariantID = v.ID
				item.VariantName = v.Name
				item.SKU = v.SKU
			} else {
				if err := s.products.DecreaseStock(txCtx, p.ID, ci.Quantity); err != nil {
					return err
				}
			}
			item.PriceCents = price
			item.Subtotal = price * int64(ci.Quantity)
			total += item.Subtotal
			items = append(items, item)
		}
		if total <= 0 {
			return fmt.Errorf("%w: order total must be positive", domain.ErrInvalidArgument)
		}
		order.SubtotalCents = total
		order.Items = items

		if couponCode != "" {
			if err := s.applyCoupon(txCtx, order, userID, couponCode, now); err != nil {
				return err
			}
		}
		order.TotalCents = order.SubtotalCents - order.DiscountCents
		if order.TotalCents < 0 {
			order.TotalCents = 0
		}

		if err := s.orders.Create(txCtx, order); err != nil {
			return err
		}
		if order.CouponID != "" {
			if err := s.coupons.CreateRedemption(txCtx, &domain.CouponRedemption{
				ID: s.ids.NewID(), CouponID: order.CouponID, UserID: userID,
				OrderID: order.ID, CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		// The event is enqueued in the same transaction as the order, so a crash
		// between commit and publish can never lose it.
		return s.enqueue(txCtx, SubjectOrderCreated, order)
	})
	if err != nil {
		return nil, err
	}

	// The cart is cleared only after the transaction commits. Redis is shared,
	// so the change is visible to the user's next request on any instance.
	if err := s.carts.Delete(ctx, userID); err != nil {
		s.logger.Warn("failed to clear cart after checkout", "userId", userID, "error", err)
	}

	s.invalidateProducts(ctx, order)
	s.metrics.Counter("openshop_orders_created_total", 1, map[string]string{"currency": order.Currency})
	s.metrics.Gauge("openshop_orders_pending_total", 1, nil)
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
		return s.enqueue(txCtx, SubjectOrderCancelled, o)
	})
	if err != nil {
		return nil, err
	}
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

	var out *domain.Order
	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		o, err := s.orders.FindByID(txCtx, orderID)
		if err != nil {
			return err
		}
		if o.Status == domain.OrderPaid {
			out = o // already processed
			return nil
		}
		if !o.Payable() {
			return domain.ErrOrderNotPayable
		}
		now := s.clock.Now()
		o.Status = domain.OrderPaid
		o.PaymentID = paymentID
		o.PaidAt = &now
		o.UpdatedAt = now
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		s.metrics.Counter("openshop_orders_paid_total", 1, map[string]string{"currency": o.Currency})
		return s.enqueue(txCtx, SubjectOrderPaid, o)
	})
	if err != nil {
		return nil, err
	}
	s.invalidateProducts(ctx, out)
	return out, nil
}

// MarkRefunded transitions a paid order to refunded and returns its stock. Like
// MarkPaid it is idempotent and serialised by a distributed lock.
func (s *OrderService) MarkRefunded(ctx context.Context, orderID, paymentID string) (*domain.Order, error) {
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
		if o.Status == domain.OrderRefunded {
			out = o
			return nil
		}
		if !o.Refundable() {
			return domain.ErrOrderNotRefundable
		}
		if err := s.releaseStock(txCtx, o); err != nil {
			return err
		}
		o.Status = domain.OrderRefunded
		o.PaymentID = paymentID
		o.UpdatedAt = s.clock.Now()
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		s.metrics.Counter("openshop_orders_refunded_total", 1, map[string]string{"currency": o.Currency})
		return s.enqueue(txCtx, SubjectOrderRefunded, o)
	})
	if err != nil {
		return nil, err
	}
	s.invalidateProducts(ctx, out)
	return out, nil
}

// applyCoupon validates and consumes a coupon inside the checkout transaction.
func (s *OrderService) applyCoupon(ctx context.Context, order *domain.Order, userID, code string, now time.Time) error {
	coupon, err := s.coupons.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := coupon.Validate(order.SubtotalCents, now); err != nil {
		return err
	}
	if coupon.PerUserLimit > 0 {
		used, err := s.coupons.CountRedemptions(ctx, coupon.ID, userID)
		if err != nil {
			return err
		}
		if used >= int64(coupon.PerUserLimit) {
			return domain.ErrCouponExhausted
		}
	}
	// Atomic compare-and-set on used_count enforces the global limit across all
	// replicas; it participates in this transaction and rolls back on failure.
	if err := s.coupons.IncrementUsage(ctx, coupon.ID); err != nil {
		return err
	}
	order.CouponID = coupon.ID
	order.CouponCode = coupon.Code
	order.DiscountCents = coupon.DiscountFor(order.SubtotalCents)
	return nil
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
		if item.VariantID != "" {
			if err := s.variants.IncreaseStock(ctx, item.VariantID, item.Quantity); err != nil {
				return err
			}
			continue
		}
		if err := s.products.IncreaseStock(ctx, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

// invalidateProducts evicts cached product views so stock changes are visible
// immediately. The event consumer also invalidates, which covers any replicas
// that missed this call.
func (s *OrderService) invalidateProducts(ctx context.Context, o *domain.Order) {
	if s.productCache == nil || o == nil {
		return
	}
	ids := make([]string, 0, len(o.Items))
	for _, item := range o.Items {
		ids = append(ids, item.ProductID)
	}
	if len(ids) > 0 {
		s.productCache.InvalidateProductCache(ctx, ids...)
	}
}

func (s *OrderService) enqueue(ctx context.Context, subject string, o *domain.Order) error {
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
	return s.outbox.Enqueue(ctx, port.Event{
		ID: s.ids.NewID(), Subject: subject, Payload: encodeEvent(evt),
		TraceParent: s.tracer.Inject(ctx),
	})
}

// orderNo builds a human-sortable, collision-resistant order number.
func orderNo(now time.Time, id string) string {
	suffix := id
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return "OS" + now.Format("20060102150405") + suffix
}
