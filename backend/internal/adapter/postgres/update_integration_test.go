package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// The update statements use explicit column maps, so a field that is added to
// the domain but forgotten in the map is silently dropped: the API reports
// success and the value never reaches the database. These tests set every
// mutable field, update it, and read it back, so that class of bug cannot ship.
func TestProductUpdatePersistsEveryField(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewProductRepository(db)
	now := time.Now().UTC()

	product := &domain.Product{
		ID: testUUID(t), Title: "Original", Slug: "test-" + testUUID(t),
		Description: "before", PriceCents: 1000, CostCents: 400, WeightGrams: 100,
		Currency: "CNY", Status: domain.ProductDraft, Stock: 5, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, product); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_ = db.session(context.Background()).Exec("DELETE FROM products WHERE id = ?", product.ID).Error
	})

	product.Title = "Updated"
	product.Names = map[string]string{"en": "Updated EN", "zh": "已更新"}
	product.Description = "after"
	product.PriceCents = 2000
	product.CostCents = 900
	product.WeightGrams = 250
	product.CoverImage = "/uploads/x.jpg"
	product.Images = []string{"/uploads/y.jpg"}
	product.Status = domain.ProductPublished
	product.Stock = 9
	product.UpdatedAt = now.Add(time.Minute)

	if err := repo.Update(ctx, product); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.FindByID(ctx, product.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"title", got.Title, "Updated"},
		{"names.en", got.Names["en"], "Updated EN"},
		{"names.zh", got.Names["zh"], "已更新"},
		{"description", got.Description, "after"},
		{"price", got.PriceCents, int64(2000)},
		{"cost", got.CostCents, int64(900)},
		{"weight", got.WeightGrams, 250},
		{"cover", got.CoverImage, "/uploads/x.jpg"},
		{"status", string(got.Status), "published"},
		{"stock", got.Stock, 9},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if len(got.Images) != 1 || got.Images[0] != "/uploads/y.jpg" {
		t.Errorf("images = %v", got.Images)
	}
}

func TestVariantUpdatePersistsEveryField(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	productRepo := NewProductRepository(db)
	variantRepo := NewVariantRepository(db)
	now := time.Now().UTC()

	product := &domain.Product{
		ID: testUUID(t), Title: "Host", Slug: "test-" + testUUID(t),
		PriceCents: 1000, Currency: "CNY", Status: domain.ProductDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := productRepo.Create(ctx, product); err != nil {
		t.Fatalf("create product: %v", err)
	}
	variant := &domain.Variant{
		ID: testUUID(t), ProductID: product.ID, SKU: "SKU-" + testUUID(t), Name: "Original",
		PriceCents: 1000, CostCents: 300, Stock: 1, WeightGrams: 10, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := variantRepo.Create(ctx, variant); err != nil {
		t.Fatalf("create variant: %v", err)
	}
	t.Cleanup(func() {
		_ = db.session(context.Background()).Exec("DELETE FROM products WHERE id = ?", product.ID).Error
	})

	variant.Name = "Updated"
	variant.PriceCents = 2500
	variant.CostCents = 1100
	variant.Stock = 7
	variant.WeightGrams = 90
	variant.Active = false
	variant.UpdatedAt = now.Add(time.Minute)

	if err := variantRepo.Update(ctx, variant); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := variantRepo.FindByID(ctx, variant.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"name", got.Name, "Updated"},
		{"price", got.PriceCents, int64(2500)},
		{"cost", got.CostCents, int64(1100)},
		{"stock", got.Stock, 7},
		{"weight", got.WeightGrams, 90},
		{"active", got.Active, false},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
