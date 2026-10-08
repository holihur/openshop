package domain

import "time"

// PeriodCounts buckets a counter into rolling windows so the console shows
// "day / week / fortnight / month" instead of only the current instant.
//
//	Today     - since 00:00 UTC today
//	Week      - the last 7 days
//	Fortnight - the last 14 days
//	Month     - the last 30 days
type PeriodCounts struct {
	Today     int64 `json:"today"`
	Week      int64 `json:"week"`
	Fortnight int64 `json:"fortnight"`
	Month     int64 `json:"month"`
}

// OrdersSummary is the orders module header.
type OrdersSummary struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"byStatus"`
	Created  PeriodCounts     `json:"created"`
	Paid     PeriodCounts     `json:"paid"`
	// RevenueCents is the paid revenue per window, in the settlement currency.
	RevenueCents PeriodCounts `json:"revenueCents"`
}

// CustomersSummary is the customers module header.
type CustomersSummary struct {
	Total    int64        `json:"total"`
	Disabled int64        `json:"disabled"`
	New      PeriodCounts `json:"new"`
}

// ProductsSummary is the catalog module header. Inventory is a snapshot, so it
// has no windows.
type ProductsSummary struct {
	Total      int64 `json:"total"`
	Published  int64 `json:"published"`
	Draft      int64 `json:"draft"`
	LowStock   int64 `json:"lowStock"`
	OutOfStock int64 `json:"outOfStock"`
}

// CouponsSummary is the coupons module header.
type CouponsSummary struct {
	Total       int64        `json:"total"`
	Active      int64        `json:"active"`
	Redemptions PeriodCounts `json:"redemptions"`
}

// TicketsSummary is the support module header.
type TicketsSummary struct {
	Open       int64        `json:"open"`
	Pending    int64        `json:"pending"`
	Unassigned int64        `json:"unassigned"`
	Created    PeriodCounts `json:"created"`
}

// ReturnsSummary is the returns module header.
type ReturnsSummary struct {
	Requested int64        `json:"requested"`
	Approved  int64        `json:"approved"`
	Created   PeriodCounts `json:"created"`
}

// WithdrawalsSummary is the withdrawals module header.
type WithdrawalsSummary struct {
	Requested int64        `json:"requested"`
	Approved  int64        `json:"approved"`
	Paid      int64        `json:"paid"`
	Created   PeriodCounts `json:"created"`
}

// ReviewsSummary is the reviews module header.
type ReviewsSummary struct {
	Total   int64        `json:"total"`
	Created PeriodCounts `json:"created"`
}

// CommissionsSummary is the referral commissions module header.
type CommissionsSummary struct {
	Pending  int64        `json:"pending"`
	Approved int64        `json:"approved"`
	Created  PeriodCounts `json:"created"`
}

// OpsSummary aggregates the counters each operations page shows at the top, in
// one request so a page needs a single round trip.
type OpsSummary struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	Orders      OrdersSummary      `json:"orders"`
	Customers   CustomersSummary   `json:"customers"`
	Products    ProductsSummary    `json:"products"`
	Coupons     CouponsSummary     `json:"coupons"`
	Tickets     TicketsSummary     `json:"tickets"`
	Returns     ReturnsSummary     `json:"returns"`
	Withdrawals WithdrawalsSummary `json:"withdrawals"`
	Reviews     ReviewsSummary     `json:"reviews"`
	Commissions CommissionsSummary `json:"commissions"`
}
