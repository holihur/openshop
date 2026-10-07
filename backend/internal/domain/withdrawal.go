package domain

import "time"

// WithdrawalStatus is the lifecycle of a wallet withdrawal request. Approval is
// a review step; the payout itself happens offline.
type WithdrawalStatus string

const (
	WithdrawalRequested WithdrawalStatus = "requested" // held, awaiting review
	WithdrawalApproved  WithdrawalStatus = "approved"  // reviewed, awaiting offline payout
	WithdrawalPaid      WithdrawalStatus = "paid"      // paid out offline
	WithdrawalRejected  WithdrawalStatus = "rejected"  // refused, funds returned
	WithdrawalCancelled WithdrawalStatus = "cancelled" // withdrawn by the customer
)

// Valid reports whether the status is known.
func (s WithdrawalStatus) Valid() bool {
	switch s {
	case WithdrawalRequested, WithdrawalApproved, WithdrawalPaid, WithdrawalRejected, WithdrawalCancelled:
		return true
	}
	return false
}

// Open reports whether the request still holds funds.
func (s WithdrawalStatus) Open() bool {
	return s == WithdrawalRequested || s == WithdrawalApproved
}

// Withdrawal is a customer's request to take money out of their wallet. The
// funds are debited when the request is created and returned if it is rejected
// or cancelled.
type Withdrawal struct {
	ID            string
	UserID        string
	AmountCents   int64
	Currency      string
	Method        string
	AccountName   string
	AccountNo     string
	Note          string
	Status        WithdrawalStatus
	RejectReason  string
	PaidReference string
	ReviewedBy    string
	ReviewedAt    *time.Time
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// WithdrawalFilter is a storage-agnostic query for withdrawal requests.
type WithdrawalFilter struct {
	Status   *WithdrawalStatus
	UserID   string
	Page     int
	PageSize int
}
