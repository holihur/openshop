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
	page, size := handler.ParsePage(c, 20)
	coupons, err := h.Coupons.List(c.Request.Context(), domain.CouponFilter{Page: page, PageSize: size})
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.CouponView, 0, len(coupons.Items))
	for _, coupon := range coupons.Items {
		out = append(out, handler.ToCouponView(coupon))
	}
	response.Paginated(c, out, coupons.Total, coupons.Page, coupons.PageSize)
}

type couponRedemptionView struct {
	ID            string `json:"id"`
	OrderID       string `json:"orderId"`
	OrderNo       string `json:"orderNo"`
	UserID        string `json:"userId"`
	UserEmail     string `json:"userEmail"`
	DiscountCents int64  `json:"discountCents"`
	CreatedAt     string `json:"createdAt"`
}

// ListCouponRedemptions returns a coupon's usage history (admin only).
func (h *Handler) ListCouponRedemptions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Coupons.ListRedemptions(c.Request.Context(), c.Param("id"), domain.CouponFilter{Page: page, PageSize: size})
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]couponRedemptionView, 0, len(result.Items))
	for _, r := range result.Items {
		out = append(out, couponRedemptionView{
			ID: r.ID, OrderID: r.OrderID, OrderNo: r.OrderNo, UserID: r.UserID,
			UserEmail: r.UserEmail, DiscountCents: r.DiscountCents,
			CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
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
