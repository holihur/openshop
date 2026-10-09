package domain

import (
	"strings"
	"time"
)

type ProductStatus string

const (
	ProductDraft     ProductStatus = "draft"
	ProductPublished ProductStatus = "published"
	ProductArchived  ProductStatus = "archived"
)

type Category struct {
	ID        string
	Name      string
	Names     map[string]string // locale -> localized name
	Slug      string
	ParentID  string
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LocalizedName returns the name for a locale, falling back to the base
// language subtag and then the default name.
func (c *Category) LocalizedName(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale != "" {
		if n := c.Names[locale]; n != "" {
			return n
		}
		if i := strings.IndexAny(locale, "-_"); i > 0 {
			if n := c.Names[locale[:i]]; n != "" {
				return n
			}
		}
	}
	return c.Name
}

type Product struct {
	ID         string
	CategoryID string
	Title      string
	Slug       string
	// Names holds per-locale titles; an absent locale falls back to Title.
	Names       map[string]string
	Description string
	// PriceCents stores the price in the minor currency unit to avoid floating
	// point money. Currency is an ISO-4217 code.
	PriceCents int64
	// WeightGrams is used for weight-based shipping; 0 means unset.
	WeightGrams int
	Currency    string
	CoverImage  string
	Images      []string
	Status      ProductStatus
	Stock       int
	// Variants are loaded on demand and persisted in their own table. A product
	// with no variants keeps its inventory on the product row.
	Variants []Variant
	// FAQs are loaded on demand and persisted in their own table.
	FAQs      []ProductFAQ
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *Product) Available() bool {
	return p.Status == ProductPublished && p.Stock > 0
}

// ProductFilter describes a catalog query. It is storage agnostic so the
// service layer never builds SQL.
type ProductFilter struct {
	CategoryID string
	Keyword    string
	Status     *ProductStatus
	Page       int
	PageSize   int
	Sort       string // "newest", "price_asc", "price_desc"
	// Cursor, when set, switches to keyset pagination (stable and O(1) at any
	// depth). Page is ignored in that mode.
	Cursor string
	// CursorMode requests keyset pagination even on the first page (where Cursor
	// is empty).
	CursorMode bool
}

type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
	// NextCursor is set by keyset (cursor) listings; empty for offset listings.
	NextCursor string
}
