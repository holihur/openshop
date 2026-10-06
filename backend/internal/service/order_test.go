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
	svc      *OrderService
	products *fakeProductRepo
	carts    *fakeCartRepo
	orders   *fakeOrderRepo
	coupons  *fakeCouponRepo
	outbox   *fakeOutbox
	locker   *fakeLocker
}

func newOrderFixture() *orderFixture {
	products := newFakeProductRepo()
	carts := newFakeCartRepo()
	orders := newFakeOrderRepo()
	coupons := newFakeCouponRepo()
	outbox := newFakeOutbox()
	locker := newFakeLocker()
	svc := NewOrderService(
		orders, products, coupons, carts, locker, fakeTx{}, outbox,
		&seqIDs{}, fixedClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		nopLogger{}, nil, port.NopMetrics{}, port.NoopTracer{}, 30*time.Minute, "CNY",
	)
	return &orderFixture{
		svc: svc, products: products, carts: carts, orders: orders,
		coupons: coupons, outbox: outbox, locker: locker,
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

	order, err := f.svc.Checkout(context.Background(), "u1", "")
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

	_, err := f.svc.Checkout(context.Background(), "u1", "")
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
	if _, err := f.svc.Checkout(context.Background(), "u1", ""); !errors.Is(err, domain.ErrCartEmpty) {
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

	order, err := f.svc.Checkout(context.Background(), "u1", "SAVE10")
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

	_, err := f.svc.Checkout(context.Background(), "u1", "ONCE")
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

	order, err := f.svc.Checkout(context.Background(), "u1", "")
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

	order, _ := f.svc.Checkout(context.Background(), "u1", "")
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

	order, _ := f.svc.Checkout(context.Background(), "u1", "")
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

func TestExpiredOrdersAreListed(t *testing.T) {
	f := newOrderFixture()
	seedProduct(f, "p1", 5, 100)
	cart := &domain.Cart{UserID: "u1", Items: []domain.CartItem{{ProductID: "p1", Quantity: 1}}}
	_ = f.carts.Save(context.Background(), cart)
	order, _ := f.svc.Checkout(context.Background(), "u1", "")

	future := order.ExpiresAt.Add(time.Minute)
	expired, err := f.svc.ListExpired(context.Background(), future, 10)
	if err != nil {
		t.Fatalf("list expired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expired = %d, want 1", len(expired))
	}
}
