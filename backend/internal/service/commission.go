package service

import (
	"context"
	"errors"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CommissionService implements the referral programme: a customer shares a
// code, everyone who signs up with it becomes their referee, and a share of
// each referee's completed order is paid into the referrer's wallet after a
// cooling-off period (default 15 days) so a refund can still reverse it.
type CommissionService struct {
	referrals   port.ReferralRepository
	commissions port.CommissionRepository
	wallets     *WalletService
	ids         port.IDGenerator
	clock       port.Clock
	settings    *SettingsService
}

func NewCommissionService(
	referrals port.ReferralRepository,
	commissions port.CommissionRepository,
	wallets *WalletService,
	ids port.IDGenerator,
	clock port.Clock,
	settings *SettingsService,
) *CommissionService {
	return &CommissionService{referrals: referrals, commissions: commissions, wallets: wallets, ids: ids, clock: clock, settings: settings}
}

// Enabled reports whether the referral programme is switched on.
func (s *CommissionService) Enabled(ctx context.Context) bool {
	return s.settings == nil || s.settings.Bool(ctx, "commission.enabled")
}

// EnsureCode returns the user's referral code, generating one on first use.
func (s *CommissionService) EnsureCode(ctx context.Context, userID string) (string, error) {
	code, err := s.referrals.FindCode(ctx, userID)
	if err == nil {
		return code, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return "", err
	}
	generated, err := referralCode()
	if err != nil {
		return "", err
	}
	ref, err := s.referrals.EnsureCode(ctx, userID, generated)
	if err != nil {
		return "", err
	}
	return ref.Code, nil
}

// ApplyReferral links a new account to the owner of a referral code. It is a
// no-op for an empty code so registration never fails on a missing referral.
func (s *CommissionService) ApplyReferral(ctx context.Context, code, refereeID string) error {
	code = strings.TrimSpace(code)
	if !s.Enabled(ctx) || code == "" || refereeID == "" {
		return nil
	}
	referrerID, err := s.referrals.FindReferrerByCode(ctx, code)
	if err != nil {
		return err
	}
	if referrerID == refereeID {
		return domain.ErrInvalidArgument
	}
	// A customer can only ever be referred once.
	if _, err := s.referrals.FindByReferee(ctx, refereeID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	return s.referrals.Create(ctx, &domain.Referral{
		ID: s.ids.NewID(), ReferrerID: referrerID, RefereeID: refereeID,
		Code: code, CreatedAt: s.clock.Now(),
	})
}

// CreateForOrder records a pending commission when a referred customer's order
// completes. It is idempotent: at most one commission per order.
func (s *CommissionService) CreateForOrder(ctx context.Context, order *domain.Order) error {
	if !s.Enabled(ctx) || order.UserID == "" {
		return nil
	}
	referral, err := s.referrals.FindByReferee(ctx, order.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if _, err := s.commissions.FindByOrder(ctx, order.ID); err == nil {
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	base := order.TotalCents
	if s.settings.String(ctx, "commission.base") == "subtotal" {
		base = order.SubtotalCents
	}
	rate := s.settings.Int(ctx, "commission.rate_bps")
	amount := base * int64(rate) / 10000
	if amount <= 0 {
		return nil
	}
	holdDays := s.settings.Int(ctx, "commission.hold_days")
	if holdDays < 0 {
		holdDays = 0
	}
	now := s.clock.Now()
	return s.commissions.Create(ctx, &domain.Commission{
		ID: s.ids.NewID(), ReferrerID: referral.ReferrerID, RefereeID: order.UserID,
		OrderID: order.ID, BaseCents: base, RateBps: rate, AmountCents: amount,
		Status: domain.CommissionPending, HoldUntil: now.AddDate(0, 0, holdDays), CreatedAt: now,
	})
}

// SettleDue pays out every commission whose cooling-off period has elapsed and
// returns how many were settled. It is safe to run on every replica: each row
// is claimed by an atomic status transition.
func (s *CommissionService) SettleDue(ctx context.Context, limit int) (int, error) {
	if s.wallets == nil {
		return 0, nil
	}
	due, err := s.commissions.ListDue(ctx, s.clock.Now(), limit)
	if err != nil {
		return 0, err
	}
	settled := 0
	for i := range due {
		c := due[i]
		if err := s.commissions.MarkApproved(ctx, c.ID, s.clock.Now()); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				continue // another replica settled it
			}
			return settled, err
		}
		if _, err := s.wallets.Credit(ctx, c.ReferrerID, c.AmountCents, domain.WalletCommission,
			"commission", c.ID, "Referral commission"); err != nil {
			return settled, err
		}
		settled++
	}
	return settled, nil
}

// ReverseForOrder cancels a pending commission, or claws back an already paid
// one (e.g. when the order is refunded).
func (s *CommissionService) ReverseForOrder(ctx context.Context, orderID string) error {
	c, err := s.commissions.FindByOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if c.Status == domain.CommissionReversed {
		return nil
	}
	if c.Status == domain.CommissionApproved && s.wallets != nil {
		if _, err := s.wallets.Adjust(ctx, c.ReferrerID, -c.AmountCents, "Commission reversal"); err != nil {
			return err
		}
	}
	return s.commissions.MarkReversed(ctx, c.ID)
}

// Summary reports a customer's referral code and earnings.
func (s *CommissionService) Summary(ctx context.Context, userID string) (*domain.ReferralSummary, error) {
	code, err := s.EnsureCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	referees, err := s.referrals.ListByReferrer(ctx, userID, 1, 1)
	if err != nil {
		return nil, err
	}
	pending, approved, err := s.commissions.SumByReferrer(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &domain.ReferralSummary{
		Code: code, Referrals: referees.Total, PendingCents: pending, ApprovedCents: approved,
	}, nil
}

// ListCommissions lists a customer's own commissions.
func (s *CommissionService) ListCommissions(ctx context.Context, referrerID string, page, pageSize int) (domain.Page[domain.Commission], error) {
	return s.commissions.List(ctx, domain.CommissionFilter{ReferrerID: referrerID, Page: page, PageSize: pageSize})
}

// List returns commissions for the ops console.
func (s *CommissionService) List(ctx context.Context, f domain.CommissionFilter) (domain.Page[domain.Commission], error) {
	return s.commissions.List(ctx, f)
}

// referralCode generates an 8-character alphanumeric code.
func referralCode() (string, error) {
	token, err := randomToken(16)
	if err != nil {
		return "", err
	}
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		}
		return -1
	}, token)
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}
	return strings.ToUpper(cleaned), nil
}
