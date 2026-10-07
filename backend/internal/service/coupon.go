package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CouponService manages discount coupons and previews them for the storefront.
// Redemption limits are enforced atomically by the repository at checkout.
type CouponService struct {
	coupons port.CouponRepository
	ids     port.IDGenerator
	clock   port.Clock
}

func NewCouponService(coupons port.CouponRepository, ids port.IDGenerator, clock port.Clock) *CouponService {
	return &CouponService{coupons: coupons, ids: ids, clock: clock}
}

type CreateCouponInput struct {
	Code             string
	Description      string
	DiscountType     domain.DiscountType
	DiscountValue    int64
	MinSubtotalCents int64
	MaxDiscountCents int64
	UsageLimit       int
	PerUserLimit     int
	StartsAt         *time.Time
	EndsAt           *time.Time
	Active           bool
}

func (s *CouponService) Create(ctx context.Context, in CreateCouponInput) (*domain.Coupon, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" {
		return nil, fmt.Errorf("%w: code is required", domain.ErrInvalidArgument)
	}
	switch in.DiscountType {
	case domain.DiscountPercent:
		if in.DiscountValue < 1 || in.DiscountValue > 100 {
			return nil, fmt.Errorf("%w: percent discount must be between 1 and 100", domain.ErrInvalidArgument)
		}
	case domain.DiscountFixed:
		if in.DiscountValue < 1 {
			return nil, fmt.Errorf("%w: fixed discount must be positive", domain.ErrInvalidArgument)
		}
	default:
		return nil, fmt.Errorf("%w: unknown discount type", domain.ErrInvalidArgument)
	}

	now := s.clock.Now()
	c := &domain.Coupon{
		ID: s.ids.NewID(), Code: code, Description: in.Description,
		DiscountType: in.DiscountType, DiscountValue: in.DiscountValue,
		MinSubtotalCents: in.MinSubtotalCents, MaxDiscountCents: in.MaxDiscountCents,
		UsageLimit: in.UsageLimit, PerUserLimit: in.PerUserLimit,
		StartsAt: in.StartsAt, EndsAt: in.EndsAt, Active: in.Active,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.coupons.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CouponService) List(ctx context.Context, f domain.CouponFilter) (domain.Page[domain.Coupon], error) {
	return s.coupons.List(ctx, f)
}

// ListRedemptions returns a coupon's usage history.
func (s *CouponService) ListRedemptions(ctx context.Context, couponID string, f domain.CouponFilter) (domain.Page[domain.CouponRedemption], error) {
	return s.coupons.ListRedemptions(ctx, couponID, f)
}

// UpdateCouponInput carries partial updates; nil fields are left unchanged.
type UpdateCouponInput struct {
	Description      *string
	DiscountValue    *int64
	MinSubtotalCents *int64
	MaxDiscountCents *int64
	UsageLimit       *int
	PerUserLimit     *int
	StartsAt         *time.Time
	EndsAt           *time.Time
	Active           *bool
}

func (s *CouponService) Update(ctx context.Context, id string, in UpdateCouponInput) (*domain.Coupon, error) {
	c, err := s.coupons.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Description != nil {
		c.Description = *in.Description
	}
	if in.DiscountValue != nil {
		if c.DiscountType == domain.DiscountPercent && (*in.DiscountValue < 1 || *in.DiscountValue > 100) {
			return nil, fmt.Errorf("%w: percent discount must be between 1 and 100", domain.ErrInvalidArgument)
		}
		if c.DiscountType == domain.DiscountFixed && *in.DiscountValue < 1 {
			return nil, fmt.Errorf("%w: fixed discount must be positive", domain.ErrInvalidArgument)
		}
		c.DiscountValue = *in.DiscountValue
	}
	if in.MinSubtotalCents != nil {
		c.MinSubtotalCents = *in.MinSubtotalCents
	}
	if in.MaxDiscountCents != nil {
		c.MaxDiscountCents = *in.MaxDiscountCents
	}
	if in.UsageLimit != nil {
		c.UsageLimit = *in.UsageLimit
	}
	if in.PerUserLimit != nil {
		c.PerUserLimit = *in.PerUserLimit
	}
	if in.StartsAt != nil {
		c.StartsAt = in.StartsAt
	}
	if in.EndsAt != nil {
		c.EndsAt = in.EndsAt
	}
	if in.Active != nil {
		c.Active = *in.Active
	}
	c.UpdatedAt = s.clock.Now()
	if err := s.coupons.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// CouponPreview is the result of validating a coupon against a subtotal.
type CouponPreview struct {
	Coupon        *domain.Coupon
	DiscountCents int64
	TotalCents    int64
}

// Preview validates a coupon for a given subtotal without consuming it.
func (s *CouponService) Preview(ctx context.Context, code string, subtotal int64) (*CouponPreview, error) {
	c, err := s.coupons.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if err := c.Validate(subtotal, s.clock.Now()); err != nil {
		return nil, err
	}
	discount := c.DiscountFor(subtotal)
	return &CouponPreview{Coupon: c, DiscountCents: discount, TotalCents: subtotal - discount}, nil
}
