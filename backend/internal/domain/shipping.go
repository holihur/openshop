package domain

import "time"

// ShippingMethod is a selectable delivery option with a flat rate and an
// optional free-shipping threshold.
type ShippingMethod struct {
	ID                 string
	Code               string
	Name               string
	FlatRateCents      int64
	FreeThresholdCents int64 // 0 means never free
	Active             bool
	Sort               int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CostFor returns the shipping cost for a subtotal, honouring the free
// threshold.
func (m *ShippingMethod) CostFor(subtotal int64) int64 {
	if m.FreeThresholdCents > 0 && subtotal >= m.FreeThresholdCents {
		return 0
	}
	if m.FlatRateCents < 0 {
		return 0
	}
	return m.FlatRateCents
}
