package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

type orderFixture struct {
	svc       *OrderService
	products  *fakeProductRepo
	variants  *fakeVariantRepo
	addresses *fakeAddressRepo
	shipping  *fakeShippingRepo
	carts     *fakeCartRepo
	orders    *fakeOrderRepo
	coupons   *fakeCouponRepo
	outbox    *fakeOutbox
	locker    *fakeLocker
}

func newOrderFixture() *orderFixture { return newOrderFixtureTax(0) }

func newOrderFixtureTax(taxBps int) *orderFixture {
	products := newFakeProductRepo()
	variants := newFakeVariantRepo()
	addresses := newFakeAddressRepo()
	shipping := newFakeShippingRepo()
	carts := newFakeCartRepo()
	orders := newFakeOrderRepo()
	coupons := newFakeCouponRepo()
	outbox := newFakeOutbox()
	locker := newFakeLocker()
	svc := NewOrderService(
		orders, products, coupons, variants, addresses, shipping, nil, carts, locker, fakeTx{}, outbox,
		&seqIDs{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		nopLogger{}, nil, port.NopMetrics{}, port.NoopTracer{}, taxBps, 30*time.Minute, "CNY",
	)
	return &orderFixture{
		svc: svc, products: products, variants: variants, addresses: addresses, shipping: shipping,
		carts: carts, orders: orders, coupons: coupons, outbox: outbox, locker: locker,
	}
}

func seedProduct(f *orderFixture, id string, stock int, price int64) {
	f.products.put(&domain.Product{
		ID: id, Title: "Item " + id, PriceCents: price, Currency: "CNY",
		Status: domain.ProductPublished, Stock: stock,
	})
}

func TestCheckoutReservesStockAndClearsCart(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	seedProduct(f, "p2", 3, 250)

	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{
		{ProductID: "p1", Quantity: 2},
		{ProductID: "p2", Quantity: 1},
	}}
	if err := f.carts.Save(context.Background(), cart); err != nil {
		t.Fatalf("save cart: %v", err)
	}

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.TotalCents != 2*100+250 {
		t.Fatalf("total = %d, want 450", order.TotalCents)
	}
	if len(order.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(order.Items))
	}
	if order.Status != domain.OrderPendingPayment {
		t.Fatalf("status = %s, want pending_payment", order.Status)
	}

	p1, _ := f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 3 {
		t.Fatalf("p1 stock = %d, want 3", p1.Stock)
	}
	p2, _ := f.products.FindByID(context.Background(), "p2")
	if p2.Stock != 2 {
		t.Fatalf("p2 stock = %d, want 2", p2.Stock)
	}

	remaining, _ := f.carts.Get(context.Background(), "u1")
	if len(remaining.Items) != 0 {
		t.Fatalf("cart not cleared: %d items", len(remaining.Items))
	}

	if subjects := f.outbox.subjects(); len(subjects) != 1 || subjects[0] != SubjectOrderCreated {
		t.Fatalf("events = %v, want [order.created]", subjects)
	}
}

func TestCheckoutRejectsInsufficientStock(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 1, 100)

	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 2}}}
	_ = f.carts.Save(context.Background(), cart)

	_, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}

	p1, _ := f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 1 {
		t.Fatalf("stock changed on failure: %d", p1.Stock)
	}
	if len(f.orders.data) != 0 {
		t.Fatalf("order persisted despite failure")
	}
}

func TestCheckoutRejectsEmptyCart(t *testing.T) {
	f := newOrderFixture()
	if _, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"}); !errors.Is(err, domain.ErrCartEmpty) {
		t.Fatalf("err = %v, want ErrCartEmpty", err)
	}
}

func TestCheckoutAppliesCoupon(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 1000)
	f.coupons.put(&domain.Coupon{
		ID: "c1", Code: "SAVE10", DiscountType: domain.DiscountPercent,
		DiscountValue: 10, PerUserLimit: 1, Active: true,
	})

	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 2}}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1", CouponCode: "SAVE10"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.SubtotalCents != 2000 || order.DiscountCents != 200 || order.TotalCents != 1800 {
		t.Fatalf("totals wrong: subtotal=%d discount=%d total=%d",
			order.SubtotalCents, order.DiscountCents, order.TotalCents)
	}
	if order.CouponCode != "SAVE10" {
		t.Fatalf("coupon code = %q", order.CouponCode)
	}
	if len(f.coupons.redemptions) != 1 {
		t.Fatalf("redemptions = %d, want 1", len(f.coupons.redemptions))
	}
}

func TestCheckoutRejectsExhaustedCoupon(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 10, 500)
	f.coupons.put(&domain.Coupon{
		ID: "c1", Code: "ONCE", DiscountType: domain.DiscountFixed,
		DiscountValue: 100, UsageLimit: 1, UsedCount: 1, Active: true,
	})
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	_, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1", CouponCode: "ONCE"})
	if !errors.Is(err, domain.ErrCouponExhausted) {
		t.Fatalf("err = %v, want ErrCouponExhausted", err)
	}
	if len(f.orders.data) != 0 {
		t.Fatal("order created despite exhausted coupon")
	}
}

func TestCancelRestoresStock(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 3}}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if _, err := f.svc.Cancel(context.Background(), "u1", order.ID, false); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	p1, _ := f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 5 {
		t.Fatalf("stock after cancel = %d, want 5", p1.Stock)
	}

	// Cancelling twice must not double-restore stock.
	if _, err := f.svc.Cancel(context.Background(), "u1", order.ID, false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second cancel err = %v, want ErrConflict", err)
	}
	p1, _ = f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 5 {
		t.Fatalf("stock after double cancel = %d, want 5", p1.Stock)
	}
}

func TestMarkPaidIsIdempotent(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	order, _ := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if _, err := f.svc.MarkPaid(context.Background(), order.ID, "pay-1"); err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	paid, _ := f.orders.FindByID(context.Background(), order.ID)
	if paid.Status != domain.OrderPaid || paid.PaymentID != "pay-1" {
		t.Fatalf("order not marked paid: %+v", paid)
	}
	// A duplicate webhook must be a no-op, not an error.
	if _, err := f.svc.MarkPaid(context.Background(), order.ID, "pay-1"); err != nil {
		t.Fatalf("duplicate mark paid: %v", err)
	}
}

func TestMarkRefundedRestoresStockAndIsIdempotent(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 2}}}
	_ = f.carts.Save(context.Background(), cart)

	order, _ := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	_, _ = f.svc.MarkPaid(context.Background(), order.ID, "pay-1")

	if _, err := f.svc.MarkRefunded(context.Background(), order.ID, "pay-1"); err != nil {
		t.Fatalf("refund: %v", err)
	}
	p1, _ := f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 5 {
		t.Fatalf("stock after refund = %d, want 5", p1.Stock)
	}
	// Refunding again is a no-op and must not restore stock twice.
	if _, err := f.svc.MarkRefunded(context.Background(), order.ID, "pay-1"); err != nil {
		t.Fatalf("second refund: %v", err)
	}
	p1, _ = f.products.FindByID(context.Background(), "p1")
	if p1.Stock != 5 {
		t.Fatalf("stock after double refund = %d, want 5", p1.Stock)
	}
}

func TestCheckoutWithVariantUsesVariantStockAndPrice(t *testing.T) {
	f := newOrderFixture()
	// Product itself has no stock; the variant carries inventory and price.
	seedProduct(f, "p1", 0, 1000)
	f.variants.put(&domain.Variant{
		ID: "v1", ProductID: "p1", SKU: "SKU-RED-L", Name: "Red / L",
		PriceCents: 1200, Stock: 3, Active: true,
	})

	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{
		{ProductID: "p1", VariantID: "v1", Quantity: 2},
	}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.TotalCents != 2400 {
		t.Fatalf("total = %d, want 2400 (variant price)", order.TotalCents)
	}
	if len(order.Items) != 1 || order.Items[0].VariantID != "v1" || order.Items[0].VariantName != "Red / L" {
		t.Fatalf("order item missing variant: %+v", order.Items)
	}

	v, _ := f.variants.FindByID(context.Background(), "v1")
	if v.Stock != 1 {
		t.Fatalf("variant stock = %d, want 1", v.Stock)
	}
	// The product's own stock must be untouched.
	p, _ := f.products.FindByID(context.Background(), "p1")
	if p.Stock != 0 {
		t.Fatalf("product stock changed: %d", p.Stock)
	}

	// Cancelling restores the variant stock, not the product's.
	if _, err := f.svc.Cancel(context.Background(), "u1", order.ID, false); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	v, _ = f.variants.FindByID(context.Background(), "v1")
	if v.Stock != 3 {
		t.Fatalf("variant stock after cancel = %d, want 3", v.Stock)
	}
}

func TestCheckoutRejectsInsufficientVariantStock(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 100, 1000)
	f.variants.put(&domain.Variant{ID: "v1", ProductID: "p1", Name: "Red", Stock: 1, Active: true})

	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{
		{ProductID: "p1", VariantID: "v1", Quantity: 2},
	}}
	_ = f.carts.Save(context.Background(), cart)

	if _, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"}); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
	p, _ := f.products.FindByID(context.Background(), "p1")
	if p.Stock != 100 {
		t.Fatalf("product stock changed: %d", p.Stock)
	}
}

func TestCheckoutSnapshotsShippingAddress(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	f.addresses.put(&domain.Address{ID: "a1", UserID: "u1", Recipient: "Alice", Line1: "1 Main St"})
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1", AddressID: "a1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.ShippingAddress == nil || order.ShippingAddress.Recipient != "Alice" {
		t.Fatalf("address not snapshotted: %+v", order.ShippingAddress)
	}

	// Another user's address must not be usable.
	f.addresses.put(&domain.Address{ID: "a2", UserID: "u2", Recipient: "Bob", Line1: "2 Side St"})
	_ = f.carts.Save(context.Background(), &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}})
	if _, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1", AddressID: "a2"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user address err = %v, want ErrNotFound", err)
	}
}

func TestFulfilmentLifecycle(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)
	order, _ := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})

	// Cannot ship an unpaid order.
	if _, err := f.svc.MarkShipped(context.Background(), order.ID, "TRK-1"); !errors.Is(err, domain.ErrOrderNotShippable) {
		t.Fatalf("ship unpaid err = %v, want ErrOrderNotShippable", err)
	}

	_, _ = f.svc.MarkPaid(context.Background(), order.ID, "pay-1")
	shipped, err := f.svc.MarkShipped(context.Background(), order.ID, "TRK-1")
	if err != nil {
		t.Fatalf("ship: %v", err)
	}
	if shipped.Status != domain.OrderShipped || shipped.TrackingNo != "TRK-1" || shipped.ShippedAt == nil {
		t.Fatalf("not shipped: %+v", shipped)
	}

	completed, err := f.svc.MarkCompleted(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Status != domain.OrderCompleted || completed.CompletedAt == nil {
		t.Fatalf("not completed: %+v", completed)
	}

	// Both transitions are idempotent.
	if _, err := f.svc.MarkShipped(context.Background(), order.ID, "TRK-1"); err != nil {
		t.Fatalf("idempotent ship: %v", err)
	}
	if _, err := f.svc.MarkCompleted(context.Background(), order.ID); err != nil {
		t.Fatalf("idempotent complete: %v", err)
	}

	if subjects := f.outbox.subjects(); len(subjects) < 4 {
		t.Fatalf("expected create/paid/shipped/completed events, got %v", subjects)
	}
}

func TestCheckoutAppliesShippingAndTax(t *testing.T) {
	f := newOrderFixtureTax(600) // 6% tax
	seedProduct(f, "p1", 5, 10000)
	f.shipping.put(&domain.ShippingMethod{ID: "s1", Name: "Standard", FlatRateCents: 500, Active: true})
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	// subtotal 10000, shipping 500, taxable 10500, tax 630, total 11130.
	if order.ShippingCents != 500 || order.TaxCents != 630 || order.TotalCents != 11130 {
		t.Fatalf("shipping=%d tax=%d total=%d, want 500/630/11130",
			order.ShippingCents, order.TaxCents, order.TotalCents)
	}
	if order.ShippingMethodName != "Standard" {
		t.Fatalf("shipping method = %q", order.ShippingMethodName)
	}
}

func TestFreeShippingThreshold(t *testing.T) {
	f := newOrderFixtureTax(0)
	seedProduct(f, "p1", 5, 10000)
	f.shipping.put(&domain.ShippingMethod{
		ID: "s1", Name: "Free over 50", FlatRateCents: 500, FreeThresholdCents: 5000, Active: true,
	})
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if order.ShippingCents != 0 {
		t.Fatalf("shipping = %d, want 0 (free over threshold)", order.ShippingCents)
	}
	if order.TotalCents != 10000 {
		t.Fatalf("total = %d, want 10000", order.TotalCents)
	}
}

func TestGuestCheckoutRequiresEmailAndIssuesToken(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 1000)
	cart := &domain.Cart{UserID: "guest-1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)

	// A guest order without an email is rejected.
	if _, err := f.svc.Checkout(context.Background(), CheckoutInput{Subject: "guest-1"}); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}

	order, err := f.svc.Checkout(context.Background(), CheckoutInput{Subject: "guest-1", GuestEmail: "g@example.com"})
	if err != nil {
		t.Fatalf("guest checkout: %v", err)
	}
	if order.UserID != "" || order.GuestEmail != "g@example.com" || order.AccessToken == "" {
		t.Fatalf("guest order wrong: %+v", order)
	}

	got, err := f.svc.FindByAccessToken(context.Background(), order.AccessToken)
	if err != nil || got.ID != order.ID {
		t.Fatalf("token lookup: %v", err)
	}
}

type fakeRates struct{ rate int64 }

func (f fakeRates) Rate(context.Context, string, string) (int64, error) { return f.rate, nil }

func TestCheckoutConvertsCurrency(t *testing.T) {
	products := newFakeProductRepo()
	variants := newFakeVariantRepo()
	addresses := newFakeAddressRepo()
	shipping := newFakeShippingRepo()
	carts := newFakeCartRepo()
	orders := newFakeOrderRepo()
	coupons := newFakeCouponRepo()
	outbox := newFakeOutbox()
	locker := newFakeLocker()
	svc := NewOrderService(
		orders, products, coupons, variants, addresses, shipping, fakeRates{rate: 7_000_000},
		carts, locker, fakeTx{}, outbox,
		&seqIDs{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		nopLogger{}, nil, port.NopMetrics{}, port.NoopTracer{}, 0, 30*time.Minute, "CNY",
	)
	products.put(&domain.Product{ID: "p1", Title: "Tee", PriceCents: 1000, Currency: "CNY", Status: domain.ProductPublished, Stock: 5})
	_ = carts.Save(context.Background(), &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 2}}})

	order, err := svc.Checkout(context.Background(), CheckoutInput{UserID: "u1", Currency: "USD"})
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	// 1000 CNY * 7.0 = 7000 USD-cents per unit, x2 = 14000.
	if order.Currency != "USD" || order.SubtotalCents != 14000 || order.TotalCents != 14000 {
		t.Fatalf("conversion wrong: currency=%s subtotal=%d total=%d", order.Currency, order.SubtotalCents, order.TotalCents)
	}
}

func TestExpiredOrdersAreListed(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)
	order, _ := f.svc.Checkout(context.Background(), CheckoutInput{UserID: "u1"})

	future := order.ExpiresAt.Add(time.Minute)
	expired, err := f.svc.ListExpired(context.Background(), future, 10)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired = %d, want 1", len(expired))
	}
}
