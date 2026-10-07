package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type refundRequest struct {
	Reason      string `json:"reason"`
	AmountCents int64  `json:"amountCents"`
	Restock     bool   `json:"restock"`
}

// RefundOrder reverses part or all of a paid order through its payment provider
// (admin only). AmountCents <= 0 refunds the remaining balance; Restock returns
// inventory when the order becomes fully refunded.
func (h *Handler) RefundOrder(c *gin.Context) {
	var req refundRequest
	_ = c.ShouldBindJSON(&req)

	order, err := h.Payments.Refund(c.Request.Context(), service.RefundInput{
		OrderID: c.Param("id"), AmountCents: req.AmountCents, Reason: req.Reason, Restock: req.Restock,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "order.refund", "order", order.ID, nil)
	response.OK(c, handler.ToOrderView(*order))
}

// GetOrder returns a single order (admin only).
func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.Orders.Get(c.Request.Context(), "", c.Param("id"), true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*order))
}

type shipRequest struct {
	TrackingNo string `json:"trackingNo"`
}

// ShipOrder marks a paid order as shipped (admin only).
func (h *Handler) ShipOrder(c *gin.Context) {
	var req shipRequest
	_ = c.ShouldBindJSON(&req)
	order, err := h.Orders.MarkShipped(c.Request.Context(), c.Param("id"), req.TrackingNo)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "order.ship", "order", order.ID, map[string]string{"trackingNo": req.TrackingNo})
	response.OK(c, handler.ToOrderView(*order))
}

// CompleteOrder closes a shipped order.
func (h *Handler) CompleteOrder(c *gin.Context) {
	order, err := h.Orders.MarkCompleted(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "order.complete", "order", order.ID, nil)
	response.OK(c, handler.ToOrderView(*order))
}

// ConfirmOrderPayment marks a pending payment (for example an offline/bank
// transfer) as succeeded and the order as paid (admin only).
func (h *Handler) ConfirmOrderPayment(c *gin.Context) {
	order, err := h.Payments.Confirm(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "order.confirm_payment", "order", order.ID, nil)
	response.OK(c, handler.ToOrderView(*order))
}
