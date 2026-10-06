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
	}{
		{"electronics", "Aurora Wireless Headphones", "Active noise cancelling over-ear headphones.", 89900, 120},
		{"electronics", "Nimbus Mechanical Keyboard", "75% hot-swappable mechanical keyboard.", 45900, 200},
		{"home-living", "Terra Ceramic Mug", "Hand-glazed 350ml stoneware mug.", 6900, 500},
		{"books", "The Pragmatic Coder", "A field guide to shipping reliable software.", 5900, 300},
	}
	for i, spec := range productSpecs {
		slug := fmt.Sprintf("%s-%d", slugify(spec.title), i+1)
		p := &domain.Product{
			ID: ids.NewID(), CategoryID: catIDs[spec.catSlug], Title: spec.title, Slug: slug,
			Description: spec.desc, PriceCents: spec.price, Currency: cfg.App.Currency,
			Status: domain.ProductPublished, Stock: spec.stock, CreatedAt: now, UpdatedAt: now,
		}
		if err := products.Create(ctx, p); err != nil {
			// Unique slug violation means the seed already ran.
			if errors.Is(err, domain.ErrConflict) {
				continue
			}
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
