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
	// MinDays/MaxDays express the delivery window in business days and are
	// shown to shoppers before checkout.
	MinDays   int
	MaxDays   int
	Active    bool
	Sort      int
	CreatedAt time.Time
	UpdatedAt time.Time
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
	// MinDays/MaxDays override the method's window for this zone. 0 inherits.
	MinDays int
	MaxDays int
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

// DeliveryEstimate tells a shopper when an order would arrive and how far it is
// from free shipping. It is computed server-side so the storefront, the cart and
// any future client share one definition.
type DeliveryEstimate struct {
	MethodID           string    `json:"methodId"`
	Code               string    `json:"code"`
	Name               string    `json:"name"`
	PriceCents         int64     `json:"priceCents"`
	MinDays            int       `json:"minDays"`
	MaxDays            int       `json:"maxDays"`
	Earliest           time.Time `json:"earliest"`
	Latest             time.Time `json:"latest"`
	FreeThresholdCents int64     `json:"freeThresholdCents"`
	// FreeRemainingCents is how much more the shopper must spend for free
	// shipping; 0 when the threshold is reached or there is none.
	FreeRemainingCents int64 `json:"freeRemainingCents"`
}

// AddBusinessDays returns the date that is n business days after start, skipping
// weekends. Delivery promises are made in business days, so this is the basis of
// every estimate shown to a shopper.
func AddBusinessDays(start time.Time, days int) time.Time {
	d := start
	for i := 0; i < days; {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		i++
	}
	return d
}

// DeliveryWindow resolves the delivery window for a method, allowing a zone rate
// to override it (0 inherits). The window is returned in business days.
func DeliveryWindow(method *ShippingMethod, rate *ShippingRate) (int, int) {
	minDays, maxDays := method.MinDays, method.MaxDays
	if rate != nil {
		if rate.MinDays > 0 {
			minDays = rate.MinDays
		}
		if rate.MaxDays > 0 {
			maxDays = rate.MaxDays
		}
	}
	if maxDays < minDays {
		maxDays = minDays
	}
	return minDays, maxDays
}
