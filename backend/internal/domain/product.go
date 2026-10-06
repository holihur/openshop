package domain

import "time"

type ProductStatus string

const (
	ProductDraft     ProductStatus = "draft"
	ProductPublished ProductStatus = "published"
	ProductArchived  ProductStatus = "archived"
)

type Category struct {
	ID        string
	Name      string
	Slug      string
	ParentID  string
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Product struct {
	ID          string
	CategoryID  string
	Title       string
	Slug        string
	Description string
	// PriceCents stores the price in the minor currency unit to avoid floating
	// point money. Currency is an ISO-4217 code.
	PriceCents int64
	Currency   string
	CoverImage string
	Images     []string
	Status     ProductStatus
	Stock      int
	// Variants are loaded on demand and persisted in their own table. A product
	// with no variants keeps its inventory on the product row.
	Variants  []Variant
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
}

type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}
