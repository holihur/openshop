package domain

import "time"

// WalletTransactionType classifies a wallet ledger entry.
type WalletTransactionType string

const (
	WalletTopUp      WalletTransactionType = "topup"
	WalletPurchase   WalletTransactionType = "purchase"
	WalletRefund     WalletTransactionType = "refund"
	WalletCommission WalletTransactionType = "commission"
	WalletWithdrawal WalletTransactionType = "withdrawal"
	WalletAdjustment WalletTransactionType = "adjustment"
)

// Wallet is a user's stored-value account. BalanceCents may go negative when a
// commission is reversed after it was already paid out; spending never overdraws.
type Wallet struct {
	ID           string
	UserID       string
	Currency     string
	BalanceCents int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WalletTransaction is one append-only ledger entry. AmountCents is signed:
// positive credits the wallet, negative debits it. BalanceAfter is the running
// balance, so a statement renders without replaying history.
type WalletTransaction struct {
	ID            string
	WalletID      string
	UserID        string
	Type          WalletTransactionType
	AmountCents   int64
	BalanceAfter  int64
	ReferenceType string
	ReferenceID   string
	Description   string
	CreatedAt     time.Time
}

// WalletTxFilter queries the wallet ledger for the ops console.
type WalletTxFilter struct {
	UserID   string
	Type     *WalletTransactionType
	Page     int
	PageSize int
}
