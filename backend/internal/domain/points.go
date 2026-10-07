package domain

import "time"

// PointsTransactionType classifies a loyalty-points ledger entry.
type PointsTransactionType string

const (
	PointsEarn   PointsTransactionType = "earn"
	PointsRedeem PointsTransactionType = "redeem"
	PointsRefund PointsTransactionType = "refund"
	PointsExpire PointsTransactionType = "expire"
	PointsAdjust PointsTransactionType = "adjust"
)

// PointsAccount is a user's loyalty-points balance.
type PointsAccount struct {
	ID             string
	UserID         string
	Balance        int64
	LifetimeEarned int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PointsTransaction is one append-only loyalty entry. Points is signed.
type PointsTransaction struct {
	ID            string
	UserID        string
	Type          PointsTransactionType
	Points        int64
	BalanceAfter  int64
	ReferenceType string
	ReferenceID   string
	Description   string
	CreatedAt     time.Time
}
