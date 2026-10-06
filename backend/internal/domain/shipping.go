package domain

import (
	"strings"
	"time"
)

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

// ShippingZone groups provinces that share shipping rates.
type ShippingZone struct {
	ID        string
	Name      string
	Provinces []string
	Active    bool
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Matches reports whether the zone covers a province (case-insensitive). An
// empty province list matches everywhere (a catch-all zone).
func (z *ShippingZone) Matches(province string) bool {
	if len(z.Provinces) == 0 {
		return true
	}
	province = strings.TrimSpace(strings.ToLower(province))
	for _, p := range z.Provinces {
		if strings.TrimSpace(strings.ToLower(p)) == province {
			return true
		}
	}
	return false
}

// ShippingRate is a per-zone override of a method's price. Weight surcharges are
// charged per started kilogram.
type ShippingRate struct {
	ID                 string
	ZoneID             string
	MethodID           string
	FlatRateCents      int64
	FreeThresholdCents int64
	PerKgCents         int64
}

// Cost returns the shipping cost for a subtotal and weight in grams.
func (r *ShippingRate) Cost(subtotalCents, weightGrams int64) int64 {
	if r.FreeThresholdCents > 0 && subtotalCents >= r.FreeThresholdCents {
		return 0
	}
	cost := r.FlatRateCents
	if r.PerKgCents > 0 && weightGrams > 0 {
		kg := (weightGrams + 999) / 1000
		cost += r.PerKgCents * kg
	}
	if cost < 0 {
		return 0
	}
	return cost
}
