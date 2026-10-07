package service

import (
	"context"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CreateWithdrawalInput is a customer's withdrawal request.
type CreateWithdrawalInput struct {
	UserID      string
	AmountCents int64
	Method      string
	AccountName string
	AccountNo   string
	Note        string
}

// WithdrawalService handles wallet withdrawal requests. Funds are held (debited
// from the wallet) as soon as a request is created, so they cannot be spent
// while it is open, and are returned if it is rejected or cancelled. Approval
// is only a review step: the actual payout happens offline and staff record the
// transfer reference when they mark the request paid.
type WithdrawalService struct {
	withdrawals port.WithdrawalRepository
	wallets     *WalletService
	ids         port.IDGenerator
	clock       port.Clock
	tx          port.TxManager
	settings    *SettingsService
	currency    string
}

func NewWithdrawalService(
	withdrawals port.WithdrawalRepository,
	wallets *WalletService,
	ids port.IDGenerator,
	clock port.Clock,
	tx port.TxManager,
	settings *SettingsService,
	currency string,
) *WithdrawalService {
	return &WithdrawalService{
		withdrawals: withdrawals, wallets: wallets, ids: ids, clock: clock, tx: tx,
		settings: settings, currency: currency,
	}
}

// Enabled reports whether withdrawals are switched on.
func (s *WithdrawalService) Enabled(ctx context.Context) bool {
	return s.settings == nil || s.settings.Bool(ctx, "wallet.withdrawal_enabled")
}

// Request creates a withdrawal and holds the funds. The debit and the request
// row are committed together, so the balance can never drift from the ledger.
func (s *WithdrawalService) Request(ctx context.Context, in CreateWithdrawalInput) (*domain.Withdrawal, error) {
	if !s.Enabled(ctx) {
		return nil, domain.ErrLoyaltyDisabled
	}
	if in.UserID == "" {
		return nil, domain.ErrUnauthorized
	}
	min := int64(s.settings.Int(ctx, "wallet.min_withdrawal_cents"))
	if in.AmountCents < min {
		return nil, domain.ErrInvalidArgument
	}
	accountNo := strings.TrimSpace(in.AccountNo)
	if accountNo == "" {
		return nil, domain.ErrInvalidArgument
	}
	method := strings.TrimSpace(in.Method)
	if method == "" {
		method = "bank"
	}

	now := s.clock.Now()
	withdrawal := &domain.Withdrawal{
		ID: s.ids.NewID(), UserID: in.UserID, AmountCents: in.AmountCents, Currency: s.currency,
		Method: method, AccountName: strings.TrimSpace(in.AccountName), AccountNo: accountNo,
		Note: strings.TrimSpace(in.Note), Status: domain.WithdrawalRequested,
		CreatedAt: now, UpdatedAt: now,
	}
	err := s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		if _, err := s.wallets.Debit(txCtx, in.UserID, in.AmountCents, domain.WalletWithdrawal,
			"withdrawal", withdrawal.ID, "Withdrawal request"); err != nil {
			return err
		}
		return s.withdrawals.Create(txCtx, withdrawal)
	})
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}

// ListByUser lists a customer's own requests.
func (s *WithdrawalService) ListByUser(ctx context.Context, userID string, page, pageSize int) (domain.Page[domain.Withdrawal], error) {
	return s.withdrawals.List(ctx, domain.WithdrawalFilter{UserID: userID, Page: page, PageSize: pageSize})
}

// List lists requests for the ops console.
func (s *WithdrawalService) List(ctx context.Context, f domain.WithdrawalFilter) (domain.Page[domain.Withdrawal], error) {
	return s.withdrawals.List(ctx, f)
}

// Approve marks a request reviewed and awaiting offline payout.
func (s *WithdrawalService) Approve(ctx context.Context, id, reviewerID string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.Status != domain.WithdrawalRequested {
		return nil, domain.ErrConflict
	}
	now := s.clock.Now()
	w.Status = domain.WithdrawalApproved
	w.ReviewedBy = reviewerID
	w.ReviewedAt = &now
	w.UpdatedAt = now
	if err := s.withdrawals.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Reject refuses a request and returns the held funds to the wallet.
func (s *WithdrawalService) Reject(ctx context.Context, id, reviewerID, reason string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !w.Status.Open() {
		return nil, domain.ErrConflict
	}
	now := s.clock.Now()
	w.Status = domain.WithdrawalRejected
	w.RejectReason = strings.TrimSpace(reason)
	w.ReviewedBy = reviewerID
	w.ReviewedAt = &now
	w.UpdatedAt = now
	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		if _, err := s.wallets.Credit(txCtx, w.UserID, w.AmountCents, domain.WalletRefund,
			"withdrawal", w.ID, "Withdrawal rejected"); err != nil {
			return err
		}
		return s.withdrawals.Update(txCtx, w)
	})
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Cancel lets a customer withdraw their own request before it is paid.
func (s *WithdrawalService) Cancel(ctx context.Context, id, userID string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.UserID != userID {
		return nil, domain.ErrNotFound
	}
	if w.Status != domain.WithdrawalRequested {
		return nil, domain.ErrConflict
	}
	now := s.clock.Now()
	w.Status = domain.WithdrawalCancelled
	w.UpdatedAt = now
	err = s.tx.WithinTx(ctx, func(txCtx context.Context) error {
		if _, err := s.wallets.Credit(txCtx, w.UserID, w.AmountCents, domain.WalletRefund,
			"withdrawal", w.ID, "Withdrawal cancelled"); err != nil {
			return err
		}
		return s.withdrawals.Update(txCtx, w)
	})
	if err != nil {
		return nil, err
	}
	return w, nil
}

// MarkPaid records that the money was transferred offline. The funds were
// already debited at request time, so no wallet movement happens here.
func (s *WithdrawalService) MarkPaid(ctx context.Context, id, reference string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.Status != domain.WithdrawalApproved {
		return nil, domain.ErrConflict
	}
	now := s.clock.Now()
	w.Status = domain.WithdrawalPaid
	w.PaidReference = strings.TrimSpace(reference)
	w.PaidAt = &now
	w.UpdatedAt = now
	if err := s.withdrawals.Update(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}
