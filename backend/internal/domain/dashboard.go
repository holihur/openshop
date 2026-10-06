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
