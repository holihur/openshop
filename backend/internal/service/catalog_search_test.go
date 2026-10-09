package service

import (
	"context"
	"testing"

	"github.com/holihur/openshop/internal/domain"
)

func newSearchCatalog(products *fakeProductRepo) *CatalogService {
	return NewCatalogService(nil, products, nil, nil, nil, nil, nil, "CNY", nil)
}

func TestListProductsFallsBackToFuzzyMatch(t *testing.T) {
	products := newFakeProductRepo()
	products.put(&domain.Product{ID: "p1", Title: "Lumen Desk Lamp", PriceCents: 10000})
	products.fuzzyHits = []domain.Product{{ID: "p1", Title: "Lumen Desk Lamp", PriceCents: 10000}}
	svc := newSearchCatalog(products)

	// A misspelling finds nothing exactly...
	page, err := svc.ListProducts(context.Background(), domain.ProductFilter{Keyword: "lampp"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// ...so the service retries approximately and flags the result.
	if len(page.Items) != 1 || page.Items[0].Title != "Lumen Desk Lamp" {
		t.Fatalf("fuzzy retry returned %+v", page.Items)
	}
	if !page.Fuzzy {
		t.Error("an approximate match must be flagged so the storefront can say so")
	}
}

func TestListProductsKeepsExactMatches(t *testing.T) {
	products := newFakeProductRepo()
	products.put(&domain.Product{ID: "p1", Title: "Lumen Desk Lamp", PriceCents: 10000})
	products.fuzzyHits = []domain.Product{{ID: "p2", Title: "Something Else"}}
	svc := newSearchCatalog(products)

	page, err := svc.ListProducts(context.Background(), domain.ProductFilter{Keyword: "lumen"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Lumen Desk Lamp" {
		t.Fatalf("exact match returned %+v", page.Items)
	}
	// A good result set must never be replaced by fuzzy suggestions.
	if page.Fuzzy {
		t.Error("an exact match must not be flagged as fuzzy")
	}
}

func TestListProductsWithoutKeywordIsNotFuzzy(t *testing.T) {
	products := newFakeProductRepo()
	products.fuzzyHits = []domain.Product{{ID: "p1", Title: "Anything"}}
	svc := newSearchCatalog(products)

	page, err := svc.ListProducts(context.Background(), domain.ProductFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("an empty catalog returned %+v", page.Items)
	}
	if page.Fuzzy {
		t.Error("a browse without a keyword is never fuzzy")
	}
}
