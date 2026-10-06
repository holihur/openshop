package domain

import "time"

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

// Payment records an attempt to charge an order through a provider. Provider
// is stored as a string so the domain stays independent of concrete gateways.
type Payment struct {
	ID            string
	OrderID       string
	UserID        string
	Provider      string
	ProviderRef   string
	AmountCents   int64
	Currency      string
	Status        PaymentStatus
	FailureReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
