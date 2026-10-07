package handler

import (
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// WalletView is the API shape of a stored-value balance.
type WalletView struct {
	Currency     string `json:"currency"`
	BalanceCents int64  `json:"balanceCents"`
}

func ToWalletView(w domain.Wallet) WalletView {
	return WalletView{Currency: w.Currency, BalanceCents: w.BalanceCents}
}

// WalletTxView is one wallet ledger entry.
type WalletTxView struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	AmountCents   int64     `json:"amountCents"`
	BalanceAfter  int64     `json:"balanceAfter"`
	ReferenceType string    `json:"referenceType,omitempty"`
	ReferenceID   string    `json:"referenceId,omitempty"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToWalletTxView(tx domain.WalletTransaction) WalletTxView {
	return WalletTxView{
		ID: tx.ID, Type: string(tx.Type), AmountCents: tx.AmountCents, BalanceAfter: tx.BalanceAfter,
		ReferenceType: tx.ReferenceType, ReferenceID: tx.ReferenceID, Description: tx.Description,
		CreatedAt: tx.CreatedAt,
	}
}

// PointsView is the API shape of a loyalty balance.
type PointsView struct {
	Balance        int64 `json:"balance"`
	LifetimeEarned int64 `json:"lifetimeEarned"`
}

func ToPointsView(a domain.PointsAccount) PointsView {
	return PointsView{Balance: a.Balance, LifetimeEarned: a.LifetimeEarned}
}

// PointsTxView is one loyalty ledger entry.
type PointsTxView struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Points        int64     `json:"points"`
	BalanceAfter  int64     `json:"balanceAfter"`
	ReferenceType string    `json:"referenceType,omitempty"`
	ReferenceID   string    `json:"referenceId,omitempty"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToPointsTxView(tx domain.PointsTransaction) PointsTxView {
	return PointsTxView{
		ID: tx.ID, Type: string(tx.Type), Points: tx.Points, BalanceAfter: tx.BalanceAfter,
		ReferenceType: tx.ReferenceType, ReferenceID: tx.ReferenceID, Description: tx.Description,
		CreatedAt: tx.CreatedAt,
	}
}

// CommissionView is one referral commission.
type CommissionView struct {
	ID          string     `json:"id"`
	ReferrerID  string     `json:"referrerId"`
	RefereeID   string     `json:"refereeId"`
	OrderID     string     `json:"orderId"`
	BaseCents   int64      `json:"baseCents"`
	RateBps     int        `json:"rateBps"`
	AmountCents int64      `json:"amountCents"`
	Status      string     `json:"status"`
	HoldUntil   time.Time  `json:"holdUntil"`
	ApprovedAt  *time.Time `json:"approvedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func ToCommissionView(c domain.Commission) CommissionView {
	return CommissionView{
		ID: c.ID, ReferrerID: c.ReferrerID, RefereeID: c.RefereeID, OrderID: c.OrderID,
		BaseCents: c.BaseCents, RateBps: c.RateBps, AmountCents: c.AmountCents,
		Status: string(c.Status), HoldUntil: c.HoldUntil, ApprovedAt: c.ApprovedAt,
		CreatedAt: c.CreatedAt,
	}
}

// ReferralSummaryView is a customer's referral programme overview.
type ReferralSummaryView struct {
	Code          string `json:"code"`
	Referrals     int64  `json:"referrals"`
	PendingCents  int64  `json:"pendingCents"`
	ApprovedCents int64  `json:"approvedCents"`
}

func ToReferralSummaryView(s domain.ReferralSummary) ReferralSummaryView {
	return ReferralSummaryView{
		Code: s.Code, Referrals: s.Referrals, PendingCents: s.PendingCents,
		ApprovedCents: s.ApprovedCents,
	}
}
