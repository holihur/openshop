package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

type couponPreviewRequest struct {
	Code          string `json:"code" binding:"required"`
	SubtotalCents int64  `json:"subtotalCents" binding:"gte=0"`
}

// PreviewCoupon validates a coupon against a subtotal without consuming it, so
// the storefront can show the discounted total before checkout.
func (h *Handler) PreviewCoupon(c *gin.Context) {
	var req couponPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
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
