// Command seed inserts demo data (an admin account, categories and products).
// It is idempotent: existing slugs are skipped, so it can run repeatedly.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/holihur/openshop/internal/adapter/postgres"
	"github.com/holihur/openshop/internal/adapter/security"
	"github.com/holihur/openshop/internal/adapter/storage"
	"github.com/holihur/openshop/internal/config"
	"github.com/holihur/openshop/internal/domain"
)

func main() {
	_ = godotenv.Load(".env", "../.env")

	cfg, err := config.Load()
	if err != nil {
		fatal("config", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	db, err := postgres.Open(cfg.Postgres, nil)
	if err != nil {
		fatal("connect", err)
	}
	defer db.Close()

	objectStore, err := storage.New(ctx, cfg.Storage)
	if err != nil {
		fatal("storage", err)
	}

	users := postgres.NewUserRepository(db)
	categories := postgres.NewCategoryRepository(db)
	products := postgres.NewProductRepository(db)
	shipping := postgres.NewShippingMethodRepository(db)
	hasher := security.NewBcryptHasher()
	ids := security.NewUUIDGenerator()
	now := time.Now().UTC()

	// Admin account.
	adminEmail := "admin@openshop.local"
	if _, err := users.FindByEmail(ctx, adminEmail); errors.Is(err, domain.ErrNotFound) {
		hash, err := hasher.Hash("admin12345")
		if err != nil {
			fatal("hash", err)
		}
		if err := users.Create(ctx, &domain.User{
			ID: ids.NewID(), Email: adminEmail, PasswordHash: hash, Name: "OpenShop Admin",
			Role: domain.RoleAdmin, Status: domain.UserActive,
			EmailVerified: true, EmailVerifiedAt: &now, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			fatal("create admin", err)
		}
		fmt.Println("created admin:", adminEmail, "password: admin12345")
	}

	// Demo customer (storefront realm).
	customerEmail := "customer@openshop.local"
	if _, err := users.FindByEmail(ctx, customerEmail); errors.Is(err, domain.ErrNotFound) {
		hash, err := hasher.Hash("customer12345")
		if err != nil {
			fatal("hash", err)
		}
		if err := users.Create(ctx, &domain.User{
			ID: ids.NewID(), Email: customerEmail, PasswordHash: hash, Name: "Demo Customer",
			Role: domain.RoleCustomer, Status: domain.UserActive,
			EmailVerified: true, EmailVerifiedAt: &now, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			fatal("create customer", err)
		}
		fmt.Println("created customer:", customerEmail, "password: customer12345")
	}

	// Categories.
	catSpecs := []struct{ name, slug string }{
		{"Electronics", "electronics"},
		{"Home & Living", "home-living"},
		{"Books", "books"},
	}
	catIDs := map[string]string{}
	for _, spec := range catSpecs {
		existing, err := categories.List(ctx)
		if err != nil {
			fatal("list categories", err)
		}
		var found string
		for _, c := range existing {
			if c.Slug == spec.slug {
				found = c.ID
				break
			}
		}
		if found == "" {
			id := ids.NewID()
			if err := categories.Create(ctx, &domain.Category{
				ID: id, Name: spec.name, Slug: spec.slug, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				fatal("create category", err)
			}
			found = id
			fmt.Println("created category:", spec.slug)
		}
		catIDs[spec.slug] = found
	}

	// Products.
	productSpecs := []struct {
		catSlug, title, desc string
		price                int64
		stock                int
		emoji, from, to      string
	}{
		{"electronics", "Aurora Wireless Headphones", "Active noise cancelling over-ear headphones.", 89900, 120, "🎧", "#6366f1", "#8b5cf6"},
		{"electronics", "Nimbus Mechanical Keyboard", "75% hot-swappable mechanical keyboard.", 45900, 200, "⌨️", "#0ea5e9", "#22d3ee"},
		{"home-living", "Terra Ceramic Mug", "Hand-glazed 350ml stoneware mug.", 6900, 500, "☕", "#f97316", "#f59e0b"},
		{"books", "The Pragmatic Coder", "A field guide to shipping reliable software.", 5900, 300, "📘", "#10b981", "#14b8a6"},
		{"home-living", "Lumen Desk Lamp", "Dimmable warm-LED desk lamp.", 12900, 150, "💡", "#eab308", "#f59e0b"},
		{"electronics", "Orbit Webcam", "1080p webcam with a privacy shutter.", 32900, 90, "📷", "#ef4444", "#f97316"},
		{"home-living", "Voyage Backpack", "Water-resistant 22L daypack.", 25900, 80, "🎒", "#14b8a6", "#0ea5e9"},
		{"books", "Systems Design Notes", "A practical notebook for system design.", 3900, 400, "📓", "#8b5cf6", "#ec4899"},
	}
	for i, spec := range productSpecs {
		slug := fmt.Sprintf("%s-%d", slugify(spec.title), i+1)
		// Skip products that already exist so re-running the seed is silent; but
		// backfill a cover image for products seeded before covers existed.
		if existing, err := products.FindBySlug(ctx, slug); err == nil {
			if existing.CoverImage == "" {
				url, err := uploadCover(ctx, objectStore, slug, spec.title, spec.emoji, spec.from, spec.to)
				if err != nil {
					fatal("upload cover", err)
				}
				existing.CoverImage = url
				existing.Images = []string{url}
				if err := products.Update(ctx, existing); err != nil {
					fatal("update product", err)
				}
				fmt.Println("added cover:", spec.title)
			}
			continue
		} else if !errors.Is(err, domain.ErrNotFound) {
			fatal("find product", err)
		}
		url, err := uploadCover(ctx, objectStore, slug, spec.title, spec.emoji, spec.from, spec.to)
		if err != nil {
			fatal("upload cover", err)
		}
		p := &domain.Product{
			ID: ids.NewID(), CategoryID: catIDs[spec.catSlug], Title: spec.title, Slug: slug,
			Description: spec.desc, PriceCents: spec.price, Currency: cfg.App.Currency,
			CoverImage: url, Images: []string{url},
			Status: domain.ProductPublished, Stock: spec.stock, CreatedAt: now, UpdatedAt: now,
		}
		if err := products.Create(ctx, p); err != nil {
			fatal("create product", err)
		}
		fmt.Println("created product:", spec.title)
	}

	seedShipping(ctx, shipping, ids, now)

	fmt.Println("seed complete")
}

func seedShipping(ctx context.Context, shipping *postgres.ShippingMethodRepository, ids *security.UUIDGenerator, now time.Time) {
	existing, err := shipping.List(ctx, false)
	if err != nil || len(existing) > 0 {
		return
	}
	methods := []domain.ShippingMethod{
		{Code: "standard", Name: "Standard (3-5 days)", FlatRateCents: 800, FreeThresholdCents: 9900, Active: true, Sort: 1},
		{Code: "express", Name: "Express (1-2 days)", FlatRateCents: 2500, Active: true, Sort: 2},
	}
	for _, m := range methods {
		m.ID = ids.NewID()
		m.CreatedAt, m.UpdatedAt = now, now
		if err := shipping.Create(ctx, &m); err != nil {
			continue
		}
		fmt.Println("created shipping method:", m.Name)
	}
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			lastDash = false
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32)
			lastDash = false
		default:
			if !lastDash && len(out) > 0 {
				out = append(out, '-')
				lastDash = true
			}
		}
	}
	return string(out)
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", step, err)
	os.Exit(1)
}
