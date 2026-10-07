package domain

import "time"

// CommissionStatus is the lifecycle of a referral commission.
type CommissionStatus string

const (
	// CommissionPending is held during the cooling-off period.
	CommissionPending CommissionStatus = "pending"
	// CommissionApproved has been paid into the referrer's wallet.
	CommissionApproved CommissionStatus = "approved"
	// CommissionReversed was cancelled (e.g. the order was refunded).
	CommissionReversed CommissionStatus = "reversed"
)

// Referral links a customer to the person who invited them.
type Referral struct {
	ID         string
	ReferrerID string
	RefereeID  string
	Code       string
	CreatedAt  time.Time
}

// Commission is an earning owed to a referrer for a referee's order. It is held
// for a cooling-off period (default 15 days) before being credited to the
// referrer's wallet, so a refund can reverse it before payout.
type Commission struct {
	ID          string
	ReferrerID  string
	RefereeID   string
	OrderID     string
	BaseCents   int64
	RateBps     int
	AmountCents int64
	Status      CommissionStatus
	HoldUntil   time.Time
	ApprovedAt  *time.Time
	CreatedAt   time.Time
}

// ReferralSummary is the referral programme overview for one customer.
type ReferralSummary struct {
	Code          string
	Referrals     int64
	PendingCents  int64
	ApprovedCents int64
}

// CommissionFilter queries commissions for the ops console.
type CommissionFilter struct {
	Status     *CommissionStatus
	ReferrerID string
	Page       int
	PageSize   int
}
