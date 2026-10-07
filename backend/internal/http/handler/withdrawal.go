package handler

import (
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// WithdrawalView is the API shape of a wallet withdrawal request.
type WithdrawalView struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	AmountCents   int64      `json:"amountCents"`
	Currency      string     `json:"currency"`
	Method        string     `json:"method"`
	AccountName   string     `json:"accountName"`
	AccountNo     string     `json:"accountNo"`
	Note          string     `json:"note,omitempty"`
	Status        string     `json:"status"`
	RejectReason  string     `json:"rejectReason,omitempty"`
	PaidReference string     `json:"paidReference,omitempty"`
	ReviewedAt    *time.Time `json:"reviewedAt,omitempty"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

func ToWithdrawalView(w domain.Withdrawal) WithdrawalView {
	return WithdrawalView{
		ID: w.ID, UserID: w.UserID, AmountCents: w.AmountCents, Currency: w.Currency,
		Method: w.Method, AccountName: w.AccountName, AccountNo: w.AccountNo, Note: w.Note,
		Status: string(w.Status), RejectReason: w.RejectReason, PaidReference: w.PaidReference,
		ReviewedAt: w.ReviewedAt, PaidAt: w.PaidAt, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
}
