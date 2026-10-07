package service

import (
	"context"
	"errors"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// WalletService manages a user's stored-value balance. Every balance change
// writes a ledger entry in the same transaction, so a statement always
// reconciles with the balance.
type WalletService struct {
	wallets  port.WalletRepository
	ids      port.IDGenerator
	clock    port.Clock
	settings *SettingsService
	currency string
}

func NewWalletService(wallets port.WalletRepository, ids port.IDGenerator, clock port.Clock, settings *SettingsService, currency string) *WalletService {
	return &WalletService{wallets: wallets, ids: ids, clock: clock, settings: settings, currency: currency}
}

// Enabled reports whether the wallet is switched on for this store.
func (s *WalletService) Enabled(ctx context.Context) bool {
	return s.settings == nil || s.settings.Bool(ctx, "wallet.enabled")
}

// Get returns the user's wallet, creating it on first use.
func (s *WalletService) Get(ctx context.Context, userID string) (*domain.Wallet, error) {
	wallet, err := s.wallets.FindByUser(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return s.wallets.Ensure(ctx, userID, s.currency)
	}
	return wallet, err
}

// Balance returns the user's balance, or 0 when the wallet does not exist yet.
func (s *WalletService) Balance(ctx context.Context, userID string) int64 {
	wallet, err := s.Get(ctx, userID)
	if err != nil {
		return 0
	}
	return wallet.BalanceCents
}

// Transactions lists the user's ledger, newest first.
func (s *WalletService) Transactions(ctx context.Context, userID string, page, pageSize int) (domain.Page[domain.WalletTransaction], error) {
	return s.wallets.ListTransactions(ctx, domain.WalletTxFilter{UserID: userID, Page: page, PageSize: pageSize})
}

// ListTransactions lists the ledger for the ops console.
func (s *WalletService) ListTransactions(ctx context.Context, f domain.WalletTxFilter) (domain.Page[domain.WalletTransaction], error) {
	return s.wallets.ListTransactions(ctx, f)
}

// TopUp credits the wallet (a sandbox/offline top-up; a real store would credit
// only after a gateway confirms the payment).
func (s *WalletService) TopUp(ctx context.Context, userID string, amountCents int64) (*domain.Wallet, error) {
	if !s.Enabled(ctx) {
		return nil, domain.ErrLoyaltyDisabled
	}
	min := int64(s.settings.Int(ctx, "wallet.min_topup_cents"))
	if amountCents < min {
		return nil, domain.ErrInvalidArgument
	}
	if max := int64(s.settings.Int(ctx, "wallet.max_topup_cents")); max > 0 && amountCents > max {
		return nil, domain.ErrInvalidArgument
	}
	if _, err := s.Credit(ctx, userID, amountCents, domain.WalletTopUp, "topup", "", "Wallet top-up"); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

// Credit adds funds to the wallet.
func (s *WalletService) Credit(ctx context.Context, userID string, amountCents int64, txType domain.WalletTransactionType, refType, refID, description string) (int64, error) {
	if amountCents <= 0 {
		return 0, domain.ErrInvalidArgument
	}
	return s.apply(ctx, userID, amountCents, true, txType, refType, refID, description)
}

// Debit spends from the wallet, refusing to overdraw.
func (s *WalletService) Debit(ctx context.Context, userID string, amountCents int64, txType domain.WalletTransactionType, refType, refID, description string) (int64, error) {
	if amountCents <= 0 {
		return 0, domain.ErrInvalidArgument
	}
	return s.apply(ctx, userID, -amountCents, false, txType, refType, refID, description)
}

// Adjust applies a signed correction (an operator override).
func (s *WalletService) Adjust(ctx context.Context, userID string, amountCents int64, description string) (int64, error) {
	if amountCents == 0 {
		return 0, domain.ErrInvalidArgument
	}
	return s.apply(ctx, userID, amountCents, true, domain.WalletAdjustment, "adjustment", "", description)
}

func (s *WalletService) apply(ctx context.Context, userID string, delta int64, allowNegative bool, txType domain.WalletTransactionType, refType, refID, description string) (int64, error) {
	wallet, err := s.wallets.Ensure(ctx, userID, s.currency)
	if err != nil {
		return 0, err
	}
	balance, err := s.wallets.AddBalance(ctx, userID, delta, allowNegative)
	if err != nil {
		return 0, err
	}
	if err := s.wallets.AddTransaction(ctx, &domain.WalletTransaction{
		ID: s.ids.NewID(), WalletID: wallet.ID, UserID: userID, Type: txType,
		AmountCents: delta, BalanceAfter: balance, ReferenceType: refType, ReferenceID: refID,
		Description: description, CreatedAt: s.clock.Now(),
	}); err != nil {
		return 0, err
	}
	return balance, nil
}
