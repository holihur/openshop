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
	// point money. Currency is an ISO-4217 code. CostCents is the unit cost used
	// for margin reporting; 0 means unknown.
	PriceCents int64
	CostCents  int64
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
// ProductFacets summarises the filters a shopper can apply: the price range and
// the variant attributes (colour, size, ...) that actually occur.
type ProductFacets struct {
	MinPriceCents int64            `json:"minPriceCents"`
	MaxPriceCents int64            `json:"maxPriceCents"`
	Attributes    []AttributeFacet `json:"attributes"`
}

// AttributeFacet is one attribute name with its distinct values, sorted.
type AttributeFacet struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type ProductFilter struct {
	CategoryID string
	Keyword    string
	Status     *ProductStatus
	Page       int
	PageSize   int
	Sort       string // "newest", "relevance", "price_asc", "price_desc"
	// Synonyms are alternative search terms for the keyword, supplied by the
	// service from the runtime configuration.
	Synonyms map[string][]string
	// MinPriceCents/MaxPriceCents bound the product price when set.
	MinPriceCents *int64
	MaxPriceCents *int64
	// Attributes filters to products having a variant whose attributes contain
	// every given key/value pair (e.g. {"color": "red"}).
	Attributes map[string]string
	// Fuzzy relaxes the keyword match to near-miss words. It is set by the
	// service only when the exact search returned nothing.
	Fuzzy bool
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
	// Fuzzy reports that the keyword only matched approximately, so the
	// storefront can say "no exact match, showing similar results".
	Fuzzy bool
	// NextCursor is set by keyset (cursor) listings; empty for offset listings.
	NextCursor string
}
