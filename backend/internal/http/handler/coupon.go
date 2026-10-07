package handler

import (
	"time"

	"github.com/holihur/openshop/internal/domain"
)

type CouponView struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Description      string `json:"description"`
	DiscountType     string `json:"discountType"`
	DiscountValue    int64  `json:"discountValue"`
	MinSubtotalCents int64  `json:"minSubtotalCents"`
	MaxDiscountCents int64  `json:"maxDiscountCents"`
	UsageLimit       int    `json:"usageLimit"`
	UsedCount        int    `json:"usedCount"`
	PerUserLimit     int    `json:"perUserLimit"`
	Active           bool   `json:"active"`
	StartsAt         string `json:"startsAt,omitempty"`
	EndsAt           string `json:"endsAt,omitempty"`
}

func ToCouponView(c domain.Coupon) CouponView {
	view := CouponView{
		ID: c.ID, Code: c.Code, Description: c.Description,
		DiscountType: string(c.DiscountType), DiscountValue: c.DiscountValue,
		MinSubtotalCents: c.MinSubtotalCents, MaxDiscountCents: c.MaxDiscountCents,
		UsageLimit: c.UsageLimit, UsedCount: c.UsedCount, PerUserLimit: c.PerUserLimit,
		Active: c.Active,
	}
	if c.StartsAt != nil {
		view.StartsAt = c.StartsAt.UTC().Format(time.RFC3339)
	}
	if c.EndsAt != nil {
		view.EndsAt = c.EndsAt.UTC().Format(time.RFC3339)
	}
	return view
}
