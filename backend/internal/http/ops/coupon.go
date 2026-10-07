package ops

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

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

type updateCouponRequest struct {
	Description      *string `json:"description"`
	DiscountValue    *int64  `json:"discountValue"`
	MinSubtotalCents *int64  `json:"minSubtotalCents"`
	MaxDiscountCents *int64  `json:"maxDiscountCents"`
	UsageLimit       *int    `json:"usageLimit"`
	PerUserLimit     *int    `json:"perUserLimit"`
	Active           *bool   `json:"active"`
}

func (h *Handler) CreateCoupon(c *gin.Context) {
	var req createCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	startsAt, err := parseOptionalTime(req.StartsAt)
	if err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	endsAt, err := parseOptionalTime(req.EndsAt)
	if err != nil {
		response.Fail(c, handler.WrapBind(err))
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
	h.RecordAudit(c, "coupon.create", "coupon", coupon.ID, map[string]string{"code": coupon.Code})
	response.Created(c, handler.ToCouponView(*coupon))
}

func (h *Handler) ListCoupons(c *gin.Context) {
	coupons, err := h.Coupons.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.CouponView, 0, len(coupons))
	for _, coupon := range coupons {
		out = append(out, handler.ToCouponView(coupon))
	}
	response.OK(c, out)
}

func (h *Handler) UpdateCoupon(c *gin.Context) {
	var req updateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	coupon, err := h.Coupons.Update(c.Request.Context(), c.Param("id"), service.UpdateCouponInput{
		Description: req.Description, DiscountValue: req.DiscountValue,
		MinSubtotalCents: req.MinSubtotalCents, MaxDiscountCents: req.MaxDiscountCents,
		UsageLimit: req.UsageLimit, PerUserLimit: req.PerUserLimit, Active: req.Active,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "coupon.update", "coupon", coupon.ID, map[string]string{"code": coupon.Code})
	response.OK(c, handler.ToCouponView(*coupon))
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
