package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	users        port.UserRepository
	products     port.ProductRepository
	coupons      port.CouponRepository
	variants     port.VariantRepository
	addresses    port.AddressRepository
	shipping     port.ShippingMethodRepository
	zones        port.ShippingZoneRepository
	rates        port.ExchangeRates
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
	invoices     port.InvoiceRenderer
	settings     *SettingsService
	currency     string
	wallet       *WalletService
	points       *PointsService
	commission   *CommissionService
}

func NewOrderService(
	orders port.OrderRepository,
	users port.UserRepository,
	products port.ProductRepository,
	coupons port.CouponRepository,
	variants port.VariantRepository,
	addresses port.AddressRepository,
	shipping port.ShippingMethodRepository,
	zones port.ShippingZoneRepository,
	rates port.ExchangeRates,
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
	invoices port.InvoiceRenderer,
	settings *SettingsService,
	currency string,
) *OrderService {
	if metrics == nil {
		metrics = port.NopMetrics{}
	}
	if tracer == nil {
		tracer = port.NoopTracer{}
	}
	return &OrderService{
		orders: orders, users: users, products: products, coupons: coupons, variants: variants, addresses: addresses,
		shipping: shipping, zones: zones, rates: rates, carts: carts, locker: locker, tx: tx,
		outbox: outbox, ids: ids, clock: clock, logger: logger, productCache: productCache,
		metrics: metrics, tracer: tracer, invoices: invoices, settings: settings, currency: currency,
	}
}

// SetLoyalty wires the wallet, points and referral services after construction.
// They are optional: when unset, checkout simply ignores stored value.
func (s *OrderService) SetLoyalty(wallet *WalletService, points *PointsService, commission *CommissionService) {
	s.wallet, s.points, s.commission = wallet, points, commission
}

// Checkout converts the user's cart into a pending order, reserving stock. The
// per-user lock prevents duplicate submissions from retries or double clicks;
// the transaction guarantees stock and order move together.
// Invoice renders the order as a document (PDF) for download. Only the owner
// or an admin may fetch it.
func (s *OrderService) Invoice(ctx context.Context, orderID, requesterID string, admin bool) ([]byte, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !admin && order.UserID != requesterID {
		return nil, domain.ErrForbidden
	}
	if !order.Status.Invoiceable() {
		return nil, domain.ErrInvoiceUnavailable
	}
	if s.invoices == nil {
		return nil, domain.ErrNotFound
	}

	data := port.InvoiceData{
		OrderNumber:   order.OrderNo,
		IssuedAt:      order.CreatedAt,
		Status:        string(order.Status),
		Currency:      order.Currency,
		SubtotalCents: order.SubtotalCents,
		DiscountCents: order.DiscountCents,
		ShippingCents: order.ShippingCents,
		TaxCents:      order.TaxCents,
		TotalCents:    order.TotalCents,
		Email:         order.GuestEmail,
	}
	if order.ShippingAddress != nil {
		data.Customer = order.ShippingAddress.Recipient
		data.ShippingAddress = order.ShippingAddress.OneLine()
	}
	if order.UserID != "" {
		if u, err := s.users.FindByID(ctx, order.UserID); err == nil {
			if u.Name != "" {
				data.Customer = u.Name
			}
			data.Email = u.Email
		}
	}
	for _, it := range order.Items {
		name := it.Title
		if it.VariantName != "" {
			name = fmt.Sprintf("%s (%s)", name, it.VariantName)
		}
		data.Items = append(data.Items, port.InvoiceItem{
			Name: name, SKU: it.SKU, Quantity: it.Quantity,
			UnitCents: it.PriceCents, TotalCents: it.Subtotal,
		})
	}
	return s.invoices.Render(data)
}

// CheckoutInput describes a checkout request. Coupon, address and shipping are
// optional. Subject is the cart owner (a user id or a guest id).
type CheckoutInput struct {
	UserID           string
	Subject          string
	CouponCode       string
	AddressID        string
	ShippingMethodID string
	// GuestEmail is required when there is no authenticated UserID.
	GuestEmail string
	GuestPhone string
	// ShippingAddress lets guests (and users without a saved address) supply an
	// inline delivery address, which is snapshotted onto the order.
	ShippingAddress *domain.Address
	// Currency is the settlement currency; empty means the store base currency.
	Currency string
	// UseWallet spends the customer's wallet balance; Points is the number of
	// loyalty points to redeem. Both are payment instruments applied after tax.
	UseWallet bool
	Points    int64
}

func (s *OrderService) Checkout(ctx context.Context, in CheckoutInput) (out *domain.Order, err error) {
	ctx, span := s.tracer.Start(ctx, "order.checkout",
		port.Attribute{Key: "user.id", Value: in.UserID},
		port.Attribute{Key: "coupon", Value: in.CouponCode},
		port.Attribute{Key: "address.id", Value: in.AddressID},
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	subject := in.Subject
	if subject == "" {
		subject = in.UserID
	}
	if subject == "" {
		return nil, fmt.Errorf("%w: no cart owner", domain.ErrInvalidArgument)
	}
	if in.UserID == "" && in.GuestEmail == "" {
		return nil, fmt.Errorf("%w: email is required for guest checkout", domain.ErrInvalidArgument)
	}

	// Resolve the settlement currency and its conversion rate up front.
	target := strings.ToUpper(strings.TrimSpace(in.Currency))
	if target == "" {
		target = s.currency
	}
	rate := int64(1_000_000)
	if target != s.currency {
		r, err := s.rates.Rate(ctx, s.currency, target)
		if err != nil {
			return nil, err
		}
		rate = r
	}

	lock, err := s.locker.Acquire(ctx, "lock:checkout:"+subject, 15*time.Second, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Release(ctx) }()

	cart, err := s.carts.Get(ctx, subject)
	if err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, domain.ErrCartEmpty
	}

	// Snapshot the shipping address so later edits never change the order.
	var shipping *domain.Address
	if in.AddressID != "" && in.UserID != "" {
		a, err := s.addresses.FindByID(ctx, in.AddressID)
		if err != nil {
			return nil, err
		}
		if a.UserID != in.UserID {
			return nil, domain.ErrNotFound
		}
		shipping = a
	}
	if shipping == nil && in.ShippingAddress != nil {
		shipping = in.ShippingAddress
	}

	now := s.clock.Now()
	order := &domain.Order{
		ID:              s.ids.NewID(),
		OrderNo:         orderNo(now, s.ids.NewID()),
		UserID:          in.UserID,
		GuestEmail:      in.GuestEmail,
		GuestPhone:      in.GuestPhone,
		Status:          domain.OrderPendingPayment,
		Currency:        target,
		ShippingAddress: shipping,
		ExpiresAt:       now.Add(time.Duration(s.settings.Int(ctx, "checkout.order_ttl_minutes")) * time.Minute),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if in.UserID == "" {
		token, err := randomToken(32)
		if err != nil {
			return nil, err
		}
		order.AccessToken = token
	}

	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		var total int64
		var weightGrams int64
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
			cost := p.CostCents
			itemWeight := p.WeightGrams
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
				cost = v.EffectiveCost(p.CostCents)
				if v.WeightGrams > 0 {
					itemWeight = v.WeightGrams
				}
				item.VariantID = v.ID
				item.VariantName = v.Name
				item.SKU = v.SKU
			} else {
				if err := s.products.DecreaseStock(txCtx, p.ID, ci.Quantity); err != nil {
					return err
				}
			}
			item.PriceCents = domain.Convert(price, rate)
			// Snapshot the cost so historical margin never shifts when the
			// catalogue cost is later edited.
			item.CostCents = domain.Convert(cost, rate)
			item.Subtotal = item.PriceCents * int64(ci.Quantity)
			total += item.Subtotal
			weightGrams += int64(itemWeight) * int64(ci.Quantity)
			items = append(items, item)
		}
		if total <= 0 {
			return fmt.Errorf("%w: order total must be positive", domain.ErrInvalidArgument)
		}
		order.SubtotalCents = total
		order.Items = items

		if in.CouponCode != "" {
			if err := s.applyCoupon(txCtx, order, subject, in.CouponCode, now, rate); err != nil {
				return err
			}
		}

		// Shipping: the chosen method, or the store default.
		method, err := s.resolveShipping(txCtx, in.ShippingMethodID)
		if err != nil {
			return err
		}
		if method != nil {
			order.ShippingMethodID = method.ID
			order.ShippingMethodName = method.Name
			order.ShippingCents = s.shippingCost(txCtx, method, shipping, order.SubtotalCents, weightGrams, rate)
		}

		// Tax applies to the discounted subtotal plus shipping.
		taxable := order.SubtotalCents - order.DiscountCents + order.ShippingCents
		if taxable < 0 {
			taxable = 0
		}
		order.TaxCents = taxable * int64(s.settings.Int(ctx, "checkout.tax_rate_bps")) / 10000
		gross := taxable + order.TaxCents

		// Wallet balance and loyalty points are payment instruments, applied after
		// tax. They are debited now and returned if the order is cancelled.
		if err := s.applyLoyalty(txCtx, order, in, gross); err != nil {
			return err
		}
		payable := gross - order.WalletCents - order.PointsDiscountCents
		if payable <= 0 {
			// Fully covered by wallet and/or points: the order is already paid.
			order.TotalCents = 0
			order.Status = domain.OrderPaid
			order.PaidAt = &now
		} else {
			order.TotalCents = payable
		}

		if err := s.orders.Create(txCtx, order); err != nil {
			return err
		}
		if order.CouponID != "" {
			if err := s.coupons.CreateRedemption(txCtx, &domain.CouponRedemption{
				ID: s.ids.NewID(), CouponID: order.CouponID, UserID: in.UserID,
				OrderID: order.ID, OrderNo: order.OrderNo, DiscountCents: order.DiscountCents,
				CreatedAt: now,
			}); err != nil {
				return err
			}
		}
		// The event is enqueued in the same transaction as the order, so a crash
		// between commit and publish can never lose it.
		if order.Status == domain.OrderPaid {
			if err := s.enqueue(txCtx, SubjectOrderPaid, order); err != nil {
				return err
			}
		}
		return s.enqueue(txCtx, SubjectOrderCreated, order)
	})
	if err != nil {
		return nil, err
	}

	// The cart is cleared only after the transaction commits. Redis is shared,
	// so the change is visible to the user's next request on any instance.
	if err := s.carts.Delete(ctx, subject); err != nil {
		s.logger.Warn("failed to clear cart after checkout", "subject", subject, "error", err)
	}

	s.invalidateProducts(ctx, order)
	s.metrics.Counter("openshop_orders_created_total", 1, map[string]string{"currency": order.Currency})
	s.metrics.Gauge("openshop_orders_pending_total", 1, nil)
	return order, nil
}

// FindByAccessToken resolves a guest order from its access token.
func (s *OrderService) FindByAccessToken(ctx context.Context, token string) (*domain.Order, error) {
	if token == "" {
		return nil, domain.ErrNotFound
	}
	return s.orders.FindByAccessToken(ctx, token)
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
		if err := s.refundLoyalty(txCtx, o); err != nil {
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
		if err := s.refundLoyalty(txCtx, o); err != nil {
			return err
		}
		if s.commission != nil {
			if err := s.commission.ReverseForOrder(txCtx, o.ID); err != nil {
				return err
			}
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

// ApplyRefund records a (possibly partial) refund against an order and marks it
// refunded once fully refunded. Inventory is returned only when the whole order
// is refunded and the caller asked to restock, because a refund alone does not
// imply the goods came back. Serialised by a distributed lock so concurrent
// refunds cannot overshoot the total.
func (s *OrderService) ApplyRefund(ctx context.Context, orderID string, amountCents int64, restock bool, paymentID string) (*domain.Order, error) {
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
		if !o.Refundable() {
			return domain.ErrOrderNotRefundable
		}
		if amountCents <= 0 || amountCents > o.RemainingRefundableCents() {
			return fmt.Errorf("%w: invalid refund amount", domain.ErrInvalidArgument)
		}

		o.RefundedCents += amountCents
		o.PaymentID = paymentID
		o.UpdatedAt = s.clock.Now()
		full := o.FullyRefunded()
		if full {
			o.Status = domain.OrderRefunded
			if restock {
				if err := s.releaseStock(txCtx, o); err != nil {
					return err
				}
			}
			if err := s.refundLoyalty(txCtx, o); err != nil {
				return err
			}
			if s.commission != nil {
				if err := s.commission.ReverseForOrder(txCtx, o.ID); err != nil {
					return err
				}
			}
		}
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		s.metrics.Counter("openshop_orders_refunded_total", 1, map[string]string{"currency": o.Currency})
		if full {
			return s.enqueue(txCtx, SubjectOrderRefunded, o)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.invalidateProducts(ctx, out)
	return out, nil
}

// MarkShipped transitions a paid order to shipped and records the tracking
// number. Idempotent and serialised by a distributed lock.
func (s *OrderService) MarkShipped(ctx context.Context, orderID, trackingNo string) (*domain.Order, error) {
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
		if o.Status == domain.OrderShipped || o.Status == domain.OrderCompleted {
			out = o
			return nil
		}
		if !o.Shippable() {
			return domain.ErrOrderNotShippable
		}
		now := s.clock.Now()
		o.Status = domain.OrderShipped
		o.TrackingNo = trackingNo
		o.ShippedAt = &now
		o.UpdatedAt = now
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		s.metrics.Counter("openshop_orders_shipped_total", 1, nil)
		return s.enqueue(txCtx, SubjectOrderShipped, o)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MarkCompleted closes a shipped order. Idempotent.
func (s *OrderService) MarkCompleted(ctx context.Context, orderID string) (*domain.Order, error) {
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
		if o.Status == domain.OrderCompleted {
			out = o
			return nil
		}
		if !o.Completetable() {
			return domain.ErrOrderNotCompletable
		}
		now := s.clock.Now()
		o.Status = domain.OrderCompleted
		o.CompletedAt = &now
		o.UpdatedAt = now
		if err := s.orders.Update(txCtx, o); err != nil {
			return err
		}
		out = o
		if o.UserID != "" && s.points != nil {
			if pts := s.points.EarnForOrder(txCtx, o); pts > 0 {
				if err := s.points.Earn(txCtx, o.UserID, pts, "order", o.ID, "Order "+o.OrderNo); err != nil {
					return err
				}
			}
		}
		if s.commission != nil {
			if err := s.commission.CreateForOrder(txCtx, o); err != nil {
				return err
			}
		}
		s.metrics.Counter("openshop_orders_completed_total", 1, nil)
		return s.enqueue(txCtx, SubjectOrderCompleted, o)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// shippingCost resolves a zone rate for the destination (falling back to the
// method default) and converts it to the settlement currency.
func (s *OrderService) shippingCost(ctx context.Context, method *domain.ShippingMethod, addr *domain.Address, subtotal, weightGrams, rate int64) int64 {
	if addr != nil && addr.Province != "" && s.zones != nil {
		if zone, err := s.zones.FindByProvince(ctx, addr.Province); err == nil {
			if zoneRate, err := s.zones.FindRate(ctx, zone.ID, method.ID); err == nil {
				return domain.Convert(zoneRate.Cost(subtotal, weightGrams), rate)
			}
		}
	}
	threshold := domain.Convert(method.FreeThresholdCents, rate)
	if method.FreeThresholdCents > 0 && subtotal >= threshold {
		return 0
	}
	return domain.Convert(method.FlatRateCents, rate)
}

// resolveShipping returns the requested shipping method, or the store default
// when none is requested. It returns (nil, nil) when no method is configured.
func (s *OrderService) resolveShipping(ctx context.Context, id string) (*domain.ShippingMethod, error) {
	if id != "" {
		m, err := s.shipping.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !m.Active {
			return nil, fmt.Errorf("%w: shipping method is not available", domain.ErrInvalidArgument)
		}
		return m, nil
	}
	m, err := s.shipping.Default(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

// applyCoupon validates and consumes a coupon inside the checkout transaction.
func (s *OrderService) applyCoupon(ctx context.Context, order *domain.Order, subject, code string, now time.Time, rate int64) error {
	coupon, err := s.coupons.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if err := coupon.Validate(order.SubtotalCents, now); err != nil {
		return err
	}
	if coupon.PerUserLimit > 0 {
		used, err := s.coupons.CountRedemptions(ctx, coupon.ID, subject)
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
	// Fixed coupon amounts are defined in the base currency; convert them.
	if coupon.DiscountType == domain.DiscountFixed && rate != 1_000_000 {
		order.DiscountCents = domain.Convert(coupon.DiscountValue, rate)
		if order.DiscountCents > order.SubtotalCents {
			order.DiscountCents = order.SubtotalCents
		}
	}
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

// applyLoyalty debits wallet balance and loyalty points against an order. Both
// are payment instruments applied after tax; the amounts are recorded on the
// order so a cancellation can return them. The calls join the ambient
// transaction, so the balance change and the order commit together.
func (s *OrderService) applyLoyalty(ctx context.Context, order *domain.Order, in CheckoutInput, gross int64) error {
	if order.UserID == "" || gross <= 0 {
		return nil
	}
	remaining := gross
	if in.UseWallet && s.wallet != nil && s.wallet.Enabled(ctx) {
		applied := s.wallet.Balance(ctx, order.UserID)
		if applied > remaining {
			applied = remaining
		}
		if applied > 0 {
			if _, err := s.wallet.Debit(ctx, order.UserID, applied, domain.WalletPurchase, "order", order.ID, "Order "+order.OrderNo); err != nil {
				return err
			}
			order.WalletCents = applied
			remaining -= applied
		}
	}
	if in.Points > 0 && s.points != nil && s.points.Enabled(ctx) && remaining > 0 {
		rate := s.points.RedeemValue(ctx, 1)
		if rate <= 0 {
			rate = 1
		}
		points := in.Points
		if balance := s.points.Balance(ctx, order.UserID); points > balance {
			points = balance
		}
		if maxPoints := s.points.MaxRedeemable(ctx, remaining) / rate; points > maxPoints {
			points = maxPoints
		}
		if points > 0 {
			if err := s.points.Redeem(ctx, order.UserID, points, "order", order.ID, "Order "+order.OrderNo); err != nil {
				return err
			}
			order.PointsUsed = points
			order.PointsDiscountCents = points * rate
		}
	}
	return nil
}

// refundLoyalty returns the wallet balance and points debited at checkout.
func (s *OrderService) refundLoyalty(ctx context.Context, o *domain.Order) error {
	if o.UserID == "" {
		return nil
	}
	if o.WalletCents > 0 && s.wallet != nil {
		if _, err := s.wallet.Credit(ctx, o.UserID, o.WalletCents, domain.WalletRefund, "order", o.ID, "Order cancelled"); err != nil {
			return err
		}
	}
	if o.PointsUsed > 0 && s.points != nil {
		if err := s.points.Refund(ctx, o.UserID, o.PointsUsed, "order", o.ID, "Order cancelled"); err != nil {
			return err
		}
	}
	return nil
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
