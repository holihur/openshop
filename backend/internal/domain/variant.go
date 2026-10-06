package domain

import "time"

// Variant is a purchasable variation of a product (e.g. "Red / Large"). When a
// product has variants, inventory and pricing live on the variant; otherwise
// they live on the product itself. This lets a simple product and a
// multi-variant product share one checkout path.
type Variant struct {
	ID        string
	ProductID string
	SKU       string
	Name      string
	// PriceCents is the variant price; 0 means inherit the product price.
	PriceCents int64
	Stock      int
	Attributes map[string]string
	Sort       int
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// EffectivePrice returns the variant price, falling back to the product price.
func (v *Variant) EffectivePrice(productPrice int64) int64 {
	if v.PriceCents > 0 {
		return v.PriceCents
	}
	return productPrice
}

func (v *Variant) Available() bool { return v.Active && v.Stock > 0 }
