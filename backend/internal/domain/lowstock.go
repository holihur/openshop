package domain

// LowStockItem is a product or variant whose inventory has fallen to or below
// the configured threshold.
type LowStockItem struct {
	Type        string // "product" | "variant"
	ID          string
	ProductID   string
	Title       string
	VariantName string
	SKU         string
	Stock       int
}
