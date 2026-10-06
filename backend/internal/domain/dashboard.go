package domain

// Dashboard is the merchant overview shown on the admin landing page.
type Dashboard struct {
	RevenueCents    int64
	PaidOrders      int64
	PendingOrders   int64
	CancelledOrders int64
	TotalOrders     int64
	TotalProducts   int64
	TotalUsers      int64
	RecentOrders    []Order
}
