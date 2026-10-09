package domain

import "time"

// PeriodCounts buckets a counter into windows so the console shows
// "day / week / fortnight / month" instead of only the current instant.
//
//	Today     - since 00:00 UTC today
//	Yesterday - the whole previous calendar day (00:00 to 00:00 UTC)
//	Week      - the last 7 days, today included
//	Fortnight - the last 14 days
//	Month     - the last 30 days
type PeriodCounts struct {
	Today     int64 `json:"today"`
	Yesterday int64 `json:"yesterday"`
	Week      int64 `json:"week"`
	Fortnight int64 `json:"fortnight"`
	Month     int64 `json:"month"`
}

// PeriodMetric pairs each window with the equivalent window immediately before
// it, so the client can show the change without a second request:
//
//	Today     vs yesterday
//	Yesterday vs the day before
//	Week      vs the previous 7 days
//	Fortnight vs the previous 14 days
//	Month     vs the previous 30 days
type PeriodMetric struct {
	Current  PeriodCounts `json:"current"`
	Previous PeriodCounts `json:"previous"`
}

// Change is the signed difference between two windows of a PeriodMetric.
func (m PeriodMetric) Change(window string) int64 {
	switch window {
	case "today":
		return m.Current.Today - m.Previous.Today
	case "yesterday":
		return m.Current.Yesterday - m.Previous.Yesterday
	case "week":
		return m.Current.Week - m.Previous.Week
	case "fortnight":
		return m.Current.Fortnight - m.Previous.Fortnight
	case "month":
		return m.Current.Month - m.Previous.Month
	}
	return 0
}

// DailyPoint is one day of a trend series (oldest first).
type DailyPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// OrdersSummary is the orders module header.
type OrdersSummary struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"byStatus"`
	Created  PeriodMetric     `json:"created"`
	Paid     PeriodMetric     `json:"paid"`
	// RevenueCents is the paid revenue per window, in the settlement currency.
	RevenueCents PeriodMetric `json:"revenueCents"`
	// CostCents is the cost of the goods sold and ProfitCents is the gross
	// profit (revenue minus cost). The margin is profit over revenue.
	CostCents   PeriodMetric `json:"costCents"`
	ProfitCents PeriodMetric `json:"profitCents"`
	// Series is the last 30 days of created orders, for the trend chart.
	Series []DailyPoint `json:"series"`
	// RevenueSeries is the same window for paid revenue.
	RevenueSeries []DailyPoint `json:"revenueSeries"`
	// ProfitSeries is the same window for gross profit.
	ProfitSeries []DailyPoint `json:"profitSeries"`
}

// CustomersSummary is the customers module header.
type CustomersSummary struct {
	Total    int64        `json:"total"`
	Disabled int64        `json:"disabled"`
	New      PeriodMetric `json:"new"`
	Series   []DailyPoint `json:"series"`
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
	Redemptions PeriodMetric `json:"redemptions"`
	Series      []DailyPoint `json:"series"`
}

// TicketsSummary is the support module header.
type TicketsSummary struct {
	Open       int64        `json:"open"`
	Pending    int64        `json:"pending"`
	Unassigned int64        `json:"unassigned"`
	Created    PeriodMetric `json:"created"`
	Series     []DailyPoint `json:"series"`
}

// ReturnsSummary is the returns module header.
type ReturnsSummary struct {
	Requested int64        `json:"requested"`
	Approved  int64        `json:"approved"`
	Created   PeriodMetric `json:"created"`
	Series    []DailyPoint `json:"series"`
}

// WithdrawalsSummary is the withdrawals module header.
type WithdrawalsSummary struct {
	Requested int64        `json:"requested"`
	Approved  int64        `json:"approved"`
	Paid      int64        `json:"paid"`
	Created   PeriodMetric `json:"created"`
	Series    []DailyPoint `json:"series"`
}

// ReviewsSummary is the reviews module header.
type ReviewsSummary struct {
	Total   int64        `json:"total"`
	Created PeriodMetric `json:"created"`
	Series  []DailyPoint `json:"series"`
}

// CommissionsSummary is the referral commissions module header.
type CommissionsSummary struct {
	Pending  int64        `json:"pending"`
	Approved int64        `json:"approved"`
	Created  PeriodMetric `json:"created"`
	Series   []DailyPoint `json:"series"`
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
