package domain

import "time"

// NotificationType groups notifications for display and filtering.
type NotificationType string

const (
	NotificationSystem     NotificationType = "system"
	NotificationOrder      NotificationType = "order"
	NotificationTicket     NotificationType = "ticket"
	NotificationWallet     NotificationType = "wallet"
	NotificationCommission NotificationType = "commission"
	NotificationWithdrawal NotificationType = "withdrawal"
)

// Notification is a short in-app message addressed to one user. Link is a
// storefront path the client can navigate to (e.g. "/orders/abc"). Title and
// Body are an English fallback; Data carries the template variables so the
// client can render the message in the reader's own language.
type Notification struct {
	ID     string
	UserID string
	Type   NotificationType
	Title  string
	Body   string
	Link   string
	// Data is a JSON object of template variables (may be empty).
	Data      string
	ReadAt    *time.Time
	CreatedAt time.Time
}

// Read reports whether the notification has been read.
func (n Notification) Read() bool { return n.ReadAt != nil }

// NotificationFilter is a storage-agnostic query for a user's inbox.
type NotificationFilter struct {
	UserID     string
	UnreadOnly bool
	Page       int
	PageSize   int
}
