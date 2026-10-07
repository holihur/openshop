package domain

import "time"

// Refund is a single (possibly partial) refund against an order. Restock
// records whether inventory was returned to the catalog.
type Refund struct {
	ID          string
	OrderID     string
	PaymentID   string
	AmountCents int64
	Reason      string
	Restock     bool
	CreatedAt   time.Time
}

// ReturnStatus is the lifecycle of a return request.
type ReturnStatus string

const (
	ReturnRequested ReturnStatus = "requested"
	ReturnApproved  ReturnStatus = "approved"
	ReturnRejected  ReturnStatus = "rejected"
)

// ReturnRequest is a customer's request to send a delivered order back.
type ReturnRequest struct {
	ID        string
	OrderID   string
	UserID    string
	Reason    string
	Status    ReturnStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReturnFilter is a storage-agnostic query for return requests.
type ReturnFilter struct {
	Status   *ReturnStatus
	Page     int
	PageSize int
}
