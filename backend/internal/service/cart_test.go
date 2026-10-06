package service

import (
	"context"
	"errors"
	"testing"

	"github.com/holihur/openshop/internal/domain"
)

func TestCartAddVariantSnapshotsVariantPrice(t *testing.T) {
	products := newFakeProductRepo()
	variants := newFakeVariantRepo()
	carts := newFakeCartRepo()
	svc := NewCartService(carts, products, variants)

	products.put(&domain.Product{
		ID: "p1", Title: "Tee", PriceCents: 1000, Currency: "CNY",
		Status: domain.ProductPublished, Stock: 50,
	})
	variants.put(&domain.Variant{
		ID: "v1", ProductID: "p1", Name: "Red / L", PriceCents: 1200, Stock: 4, Active: true,
	})

	cart, err := svc.AddItem(context.Background(), "u1", "p1", "v1", 2)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(cart.Items))
	}
	item := cart.Items[0]
	if item.VariantID != "v1" || item.VariantName != "Red / L" {
		t.Fatalf("variant not recorded: %+v", item)
	}
	if item.PriceCents != 1200 {
		t.Fatalf("price = %d, want 1200 (variant price)", item.PriceCents)
	}
	if cart.TotalCents() != 2400 {
		t.Fatalf("total = %d, want 2400", cart.TotalCents())
	}
}

func TestCartRejectsVariantStockOverflow(t *testing.T) {
	products := newFakeProductRepo()
	variants := newFakeVariantRepo()
	carts := newFakeCartRepo()
	svc := NewCartService(carts, products, variants)

	products.put(&domain.Product{ID: "p1", Title: "Tee", PriceCents: 1000, Status: domain.ProductPublished, Stock: 50})
	variants.put(&domain.Variant{ID: "v1", ProductID: "p1", Name: "Red", Stock: 1, Active: true})

	if _, err := svc.AddItem(context.Background(), "u1", "p1", "v1", 2); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("err = %v, want ErrInsufficientStock", err)
	}
}

func TestCartTreatsVariantAndProductLinesSeparately(t *testing.T) {
	products := newFakeProductRepo()
	variants := newFakeVariantRepo()
	carts := newFakeCartRepo()
	svc := NewCartService(carts, products, variants)

	products.put(&domain.Product{ID: "p1", Title: "Tee", PriceCents: 1000, Status: domain.ProductPublished, Stock: 50})
	variants.put(&domain.Variant{ID: "v1", ProductID: "p1", Name: "Red", Stock: 5, Active: true})

	if _, err := svc.AddItem(context.Background(), "u1", "p1", "", 1); err != nil {
		t.Fatalf("add plain: %v", err)
	}
	cart, err := svc.AddItem(context.Background(), "u1", "p1", "v1", 1)
	if err != nil {
		t.Fatalf("add variant: %v", err)
	}
	if len(cart.Items) != 2 {
		t.Fatalf("items = %d, want 2 (product + variant are distinct lines)", len(cart.Items))
	}
}
