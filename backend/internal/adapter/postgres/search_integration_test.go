package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// seedSearchCatalog inserts a small published catalog and returns the product
// ids by handle.
func seedSearchCatalog(t *testing.T, db *DB) map[string]string {
	t.Helper()
	ctx := context.Background()
	// The tests share one database, so every title carries a per-run marker and
	// the assertions only look at the rows this run created.
	marker := " " + testUUID(t)[:8]
	products := NewProductRepository(db)
	variants := NewVariantRepository(db)
	now := time.Now().UTC()
	ids := map[string]string{}

	mk := func(title string, price int64) string {
		id := testUUID(t)
		p := &domain.Product{
			ID: id, Title: title + marker, Slug: "search-" + id, Description: "a test product",
			PriceCents: price, Currency: "CNY", Status: domain.ProductPublished,
			Stock: 10, CreatedAt: now, UpdatedAt: now,
		}
		if err := products.Create(ctx, p); err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		ids[title] = id
		return id
	}
	mk("Lumen Desk Lamp", 10000)
	mk("Ceramic Mug", 5000)
	mk("Draft Lamp", 7000)

	// A variant carrying attributes, for the attribute filter and the facets.
	mugID := ids["Ceramic Mug"]
	if err := variants.Create(ctx, &domain.Variant{
		ID: testUUID(t), ProductID: mugID, SKU: "MUG-RED-" + testUUID(t), Name: "Red",
		PriceCents: 5000, Stock: 3, Active: true, Attributes: map[string]string{"color": "red"},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create variant: %v", err)
	}
	// The draft product must never leak into a public search.
	if err := products.Update(ctx, &domain.Product{
		ID: ids["Draft Lamp"], Title: "Draft Lamp", Slug: "search-" + ids["Draft Lamp"],
		Description: "a test product", PriceCents: 7000, Currency: "CNY",
		Status: domain.ProductDraft, Stock: 1, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("update draft: %v", err)
	}

	t.Cleanup(func() {
		for _, id := range ids {
			// Variants reference the product, so they go first.
			_ = db.session(context.Background()).Exec("DELETE FROM product_variants WHERE product_id = ?", id).Error
			_ = db.session(context.Background()).Exec("DELETE FROM products WHERE id = ?", id).Error
		}
	})
	return ids
}

func titles(items []domain.Product) []string {
	out := make([]string, 0, len(items))
	for _, p := range items {
		out = append(out, p.Title)
	}
	return out
}

func publishedFilter() domain.ProductFilter {
	status := domain.ProductPublished
	return domain.ProductFilter{Status: &status, PageSize: 20}
}

func TestProductSearchStemsAndIgnoresDrafts(t *testing.T) {
	db := openTestDB(t)
	seedSearchCatalog(t, db)
	repo := NewProductRepository(db)

	// "lamps" must match "Lamp" through the english stemmer, and the draft
	// product must stay hidden.
	f := publishedFilter()
	f.Keyword = "lamps"
	page, err := repo.List(context.Background(), f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Lumen Desk Lamp") {
		t.Errorf("stemmed search = %v, want [Lumen Desk Lamp ...]", got)
	}

	// A partial word still matches through the substring fallback.
	f.Keyword = "ceram"
	page, err = repo.List(context.Background(), f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Ceramic Mug") {
		t.Errorf("substring search = %v, want [Ceramic Mug ...]", got)
	}
}

func TestProductSearchSynonymExpansion(t *testing.T) {
	db := openTestDB(t)
	seedSearchCatalog(t, db)
	repo := NewProductRepository(db)

	// "tumbler" appears nowhere, but is configured as a synonym of "mug".
	f := publishedFilter()
	f.Keyword = "tumbler"
	f.Synonyms = map[string][]string{"tumbler": {"mug"}}
	page, err := repo.List(context.Background(), f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Ceramic Mug") {
		t.Errorf("synonym search = %v, want [Ceramic Mug ...]", got)
	}
}

func TestProductSearchFuzzyFallbackMatchesMisspellings(t *testing.T) {
	db := openTestDB(t)
	seedSearchCatalog(t, db)
	repo := NewProductRepository(db)

	// An exact search for a misspelling finds nothing...
	f := publishedFilter()
	f.Keyword = "lampp"
	page, err := repo.List(context.Background(), f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("exact search for a misspelling returned %v", titles(page.Items))
	}

	// ...and the fuzzy pass then suggests the closest title.
	f.Fuzzy = true
	page, err = repo.List(context.Background(), f)
	if err != nil {
		t.Fatalf("fuzzy list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Lumen Desk Lamp") {
		t.Errorf("fuzzy search = %v, want [Lumen Desk Lamp ...]", got)
	}
}

func TestProductFilterByPriceAndAttributes(t *testing.T) {
	db := openTestDB(t)
	seedSearchCatalog(t, db)
	repo := NewProductRepository(db)
	ctx := context.Background()

	// Price range: only the mug (5000) falls inside 4000..6000.
	min, max := int64(4000), int64(6000)
	f := publishedFilter()
	f.MinPriceCents, f.MaxPriceCents = &min, &max
	page, err := repo.List(ctx, f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Ceramic Mug") {
		t.Errorf("price filter = %v, want [Ceramic Mug ...]", got)
	}

	// Attribute filter: only the mug has a red variant.
	f = publishedFilter()
	f.Attributes = map[string]string{"color": "red"}
	page, err = repo.List(ctx, f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := titles(page.Items); len(got) != 1 || !strings.HasPrefix(got[0], "Ceramic Mug") {
		t.Errorf("attribute filter = %v, want [Ceramic Mug ...]", got)
	}

	// A value nobody carries matches nothing.
	f.Attributes = map[string]string{"color": "chartreuse"}
	page, err = repo.List(ctx, f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("unknown attribute value matched %v", titles(page.Items))
	}
}

func TestProductFacetsReportPriceAndAttributes(t *testing.T) {
	db := openTestDB(t)
	seedSearchCatalog(t, db)
	repo := NewProductRepository(db)

	facets, err := repo.Facets(context.Background(), "")
	if err != nil {
		t.Fatalf("facets: %v", err)
	}
	// The seeded range must be contained in the reported range (other tests
	// share the database, so it can only be wider).
	if facets.MinPriceCents > 5000 || facets.MaxPriceCents < 10000 {
		t.Errorf("price range = %d..%d, want it to cover 5000..10000", facets.MinPriceCents, facets.MaxPriceCents)
	}
	var color *domain.AttributeFacet
	for i := range facets.Attributes {
		if facets.Attributes[i].Name == "color" {
			color = &facets.Attributes[i]
		}
	}
	if color == nil {
		t.Fatalf("no 'color' facet in %+v", facets.Attributes)
	}
	found := false
	for _, value := range color.Values {
		if value == "red" {
			found = true
		}
	}
	if !found {
		t.Errorf("color values = %v, want it to include red", color.Values)
	}
}
