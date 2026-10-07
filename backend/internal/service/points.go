package service

import (
	"context"
	"errors"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// PointsService manages a user's loyalty balance: points are earned when an
// order completes and redeemed for a discount at checkout.
type PointsService struct {
	points   port.PointsRepository
	ids      port.IDGenerator
	clock    port.Clock
	settings *SettingsService
}

func NewPointsService(points port.PointsRepository, ids port.IDGenerator, clock port.Clock, settings *SettingsService) *PointsService {
	return &PointsService{points: points, ids: ids, clock: clock, settings: settings}
}

// Enabled reports whether loyalty points are switched on.
func (s *PointsService) Enabled(ctx context.Context) bool {
	return s.settings == nil || s.settings.Bool(ctx, "points.enabled")
}

// Get returns the user's points account, creating it on first use.
func (s *PointsService) Get(ctx context.Context, userID string) (*domain.PointsAccount, error) {
	account, err := s.points.FindByUser(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return s.points.Ensure(ctx, userID)
	}
	return account, err
}

// Balance returns the user's points, or 0 when the account does not exist yet.
func (s *PointsService) Balance(ctx context.Context, userID string) int64 {
	account, err := s.Get(ctx, userID)
	if err != nil {
		return 0
	}
	return account.Balance
}

// Transactions lists the user's points ledger, newest first.
func (s *PointsService) Transactions(ctx context.Context, userID string, page, pageSize int) (domain.Page[domain.PointsTransaction], error) {
	return s.points.ListTransactions(ctx, userID, page, pageSize)
}

// EarnForOrder computes the points an order is worth. One point is granted per
// whole currency unit by default (points.earn_per_unit).
func (s *PointsService) EarnForOrder(ctx context.Context, order *domain.Order) int64 {
	if !s.Enabled(ctx) {
		return 0
	}
	perUnit := int64(s.settings.Int(ctx, "points.earn_per_unit"))
	if perUnit <= 0 || order.TotalCents <= 0 {
		return 0
	}
	return (order.TotalCents / 100) * perUnit
}

// Earn credits points for an order.
func (s *PointsService) Earn(ctx context.Context, userID string, points int64, refType, refID, description string) error {
	if points <= 0 {
		return nil
	}
	return s.apply(ctx, userID, points, domain.PointsEarn, refType, refID, description)
}

// Redeem spends points.
func (s *PointsService) Redeem(ctx context.Context, userID string, points int64, refType, refID, description string) error {
	if points <= 0 {
		return nil
	}
	if _, err := s.Get(ctx, userID); err != nil {
		return err
	}
	return s.apply(ctx, userID, -points, domain.PointsRedeem, refType, refID, description)
}

// Refund returns points that were redeemed against an order that was then
// cancelled. Unlike Earn it does not count toward the lifetime total.
func (s *PointsService) Refund(ctx context.Context, userID string, points int64, refType, refID, description string) error {
	if points <= 0 {
		return nil
	}
	return s.apply(ctx, userID, points, domain.PointsRefund, refType, refID, description)
}

// Adjust applies a signed correction (an operator override).
func (s *PointsService) Adjust(ctx context.Context, userID string, points int64, description string) error {
	if points == 0 {
		return domain.ErrInvalidArgument
	}
	return s.apply(ctx, userID, points, domain.PointsAdjust, "adjustment", "", description)
}

// RedeemValue converts points to a discount in cents.
func (s *PointsService) RedeemValue(ctx context.Context, points int64) int64 {
	if points <= 0 {
		return 0
	}
	return points * int64(s.settings.Int(ctx, "points.redeem_cents_per_point"))
}

// MaxRedeemable caps a points discount at a percentage of the payable amount.
func (s *PointsService) MaxRedeemable(ctx context.Context, payableCents int64) int64 {
	percent := int64(s.settings.Int(ctx, "points.max_redeem_percent"))
	if percent <= 0 {
		return 0
	}
	if percent > 100 {
		percent = 100
	}
	return payableCents * percent / 100
}

func (s *PointsService) apply(ctx context.Context, userID string, delta int64, txType domain.PointsTransactionType, refType, refID, description string) error {
	// Ensure the account exists so the atomic UPDATE always matches a row; a
	// missing row would otherwise look like an overdraft.
	if _, err := s.points.Ensure(ctx, userID); err != nil {
		return err
	}
	balance, err := s.points.AddPoints(ctx, userID, delta)
	if err != nil {
		return err
	}
	return s.points.AddTransaction(ctx, &domain.PointsTransaction{
		ID: s.ids.NewID(), UserID: userID, Type: txType, Points: delta, BalanceAfter: balance,
		ReferenceType: refType, ReferenceID: refID, Description: description, CreatedAt: s.clock.Now(),
	})
}
