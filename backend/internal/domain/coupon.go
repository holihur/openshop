package domain

import "time"

// CouponRedemption records that a user consumed a coupon on an order.
type CouponRedemption struct {
	ID        string
	CouponID  string
	UserID    string
	OrderID   string
	CreatedAt time.Time
}

type DiscountType string

const (
	DiscountPercent DiscountType = "percent"
	DiscountFixed   DiscountType = "fixed"
)

// Coupon is a discount that can be applied at checkout. DiscountValue is a
// percentage (0-100) for percent coupons or an amount in minor units for fixed
// coupons.
type Coupon struct {
	ID               string
	Code             string
	Description      string
	DiscountType     DiscountType
	DiscountValue    int64
	MinSubtotalCents int64
	// MaxDiscountCents caps a percentage discount; 0 means no cap.
	MaxDiscountCents int64
	// UsageLimit is the global redemption cap; 0 means unlimited.
	UsageLimit int
	UsedCount  int
	// PerUserLimit caps redemptions per user; 0 means unlimited.
	PerUserLimit int
	StartsAt     *time.Time
	EndsAt       *time.Time
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate checks whether the coupon may be applied to a subtotal at a point in
// time. It returns ErrCouponExhausted when the global limit is reached so the
// transport layer can map it to HTTP 409.
func (c *Coupon) Validate(subtotal int64, now time.Time) error {
	if !c.Active {
		return NewError("coupon_invalid", "coupon is not active", nil)
	}
	if c.StartsAt != nil && now.Before(*c.StartsAt) {
		return NewError("coupon_invalid", "coupon is not yet valid", nil)
	}
	if c.EndsAt != nil && now.After(*c.EndsAt) {
		return NewError("coupon_invalid", "coupon has expired", nil)
	}
	if subtotal < c.MinSubtotalCents {
		return NewError("coupon_invalid", "order does not meet the coupon minimum", nil)
	}
	if c.UsageLimit > 0 && c.UsedCount >= c.UsageLimit {
		return ErrCouponExhausted
	}
	return nil
}

// DiscountFor computes the discount in minor units, never exceeding the
// subtotal.
func (c *Coupon) DiscountFor(subtotal int64) int64 {
	if subtotal <= 0 {
		return 0
	}
	var discount int64
	switch c.DiscountType {
	case DiscountPercent:
		discount = subtotal * c.DiscountValue / 100
		if c.MaxDiscountCents > 0 && discount > c.MaxDiscountCents {
			discount = c.MaxDiscountCents
		}
	default: // DiscountFixed
		discount = c.DiscountValue
	}
	if discount > subtotal {
		discount = subtotal
	}
	if discount < 0 {
		discount = 0
	}
	return discount
}
