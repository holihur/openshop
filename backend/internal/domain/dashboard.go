package domain

// Dashboard is the merchant overview shown on the admin landing page.
type Dashboard struct {
	RevenueCents int64
	// RevenueByCurrency is the paid revenue per settlement currency; RevenueCents
	// is the slice for the store base currency.
	RevenueByCurrency map[string]int64
	LowStock          []LowStockItem
	PaidOrders        int64
	PendingOrders     int64
	CancelledOrders   int64
	TotalOrders       int64
	TotalProducts     int64
	TotalUsers        int64
	RecentOrders      []Order
}

// BalanceDrift is a stored value that no longer matches the ledger behind it.
type BalanceDrift struct {
	ID      string
	OwnerID string
	// Unit is the currency code, or "points" for the loyalty ledger.
	Unit     string
	Stored   int64
	Expected int64
}

// Diff is how far the stored value is from its ledger.
func (d BalanceDrift) Diff() int64 { return d.Stored - d.Expected }

// OrderDrift is a settled order whose collected payments do not match what it
// says was charged (minus what was refunded).
type OrderDrift struct {
	ID            string `json:"id"`
	OrderNo       string `json:"orderNo"`
	ExpectedCents int64  `json:"expectedCents"`
	ActualCents   int64  `json:"actualCents"`
}

// Diff is the amount collected beyond, or short of, what was expected.
func (d OrderDrift) Diff() int64 { return d.ActualCents - d.ExpectedCents }
