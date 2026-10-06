package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type couponView struct {
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

type createCouponRequest struct {
	Code             string `json:"code" binding:"required"`
	Description      string `json:"description"`
	DiscountType     string `json:"discountType" binding:"required,oneof=percent fixed"`
	DiscountValue    int64  `json:"discountValue" binding:"required,gt=0"`
	MinSubtotalCents int64  `json:"minSubtotalCents"`
	MaxDiscountCents int64  `json:"maxDiscountCents"`
	UsageLimit       int    `json:"usageLimit"`
	PerUserLimit     int    `json:"perUserLimit"`
	StartsAt         string `json:"startsAt"`
	EndsAt           string `json:"endsAt"`
	Active           *bool  `json:"active"`
}

type couponPreviewRequest struct {
	Code          string `json:"code" binding:"required"`
	SubtotalCents int64  `json:"subtotalCents" binding:"gte=0"`
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	var req createCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	startsAt, err := parseOptionalTime(req.StartsAt)
	if err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	endsAt, err := parseOptionalTime(req.EndsAt)
	if err != nil {
		response.Fail(c, wrapBind(err))
		return
	}

	coupon, err := h.Coupons.Create(c.Request.Context(), service.CreateCouponInput{
		Code: req.Code, Description: req.Description,
		DiscountType: domain.DiscountType(req.DiscountType), DiscountValue: req.DiscountValue,
		MinSubtotalCents: req.MinSubtotalCents, MaxDiscountCents: req.MaxDiscountCents,
		UsageLimit: req.UsageLimit, PerUserLimit: req.PerUserLimit,
		StartsAt: startsAt, EndsAt: endsAt, Active: active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.audit(c, "coupon.create", "coupon", coupon.ID, map[string]string{"code": coupon.Code})
	response.Created(c, toCouponView(*coupon))
}

func (h *Handler) ListCoupons(c *gin.Context) {
	coupons, err := h.Coupons.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]couponView, 0, len(coupons))
	for _, coupon := range coupons {
		out = append(out, toCouponView(coupon))
	}
	response.OK(c, out)
}

// PreviewCoupon validates a coupon against a subtotal without consuming it, so
// the storefront can show the discounted total before checkout.
func (h *Handler) PreviewCoupon(c *gin.Context) {
	var req couponPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	preview, err := h.Coupons.Preview(c.Request.Context(), req.Code, req.SubtotalCents)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"code":          preview.Coupon.Code,
		"discountCents": preview.DiscountCents,
		"totalCents":    preview.TotalCents,
	})
}

func toCouponView(c domain.Coupon) couponView {
	view := couponView{
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

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
