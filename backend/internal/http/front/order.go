package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// CheckoutRequest is optional: callers may include a coupon, address, shipping
// method, and (for guests) an email.
type CheckoutRequest struct {
	CouponCode       string `json:"couponCode"`
	AddressID        string `json:"addressId"`
	ShippingMethodID string `json:"shippingMethodId"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Currency         string `json:"currency"`
	// Address is an inline shipping address (used by guests).
	Address *addressRequest `json:"address"`
}

func (h *Handler) Checkout(c *gin.Context) {
	var req CheckoutRequest
	// The body is optional, so binding failures are ignored.
	_ = c.ShouldBindJSON(&req)

	order, err := h.Orders.Checkout(c.Request.Context(), service.CheckoutInput{
		UserID:           middleware.UserID(c),
		Subject:          middleware.Subject(c),
		CouponCode:       req.CouponCode,
		AddressID:        req.AddressID,
		ShippingMethodID: req.ShippingMethodID,
		GuestEmail:       req.Email,
		GuestPhone:       req.Phone,
		Currency:         req.Currency,
		ShippingAddress:  inlineAddress(req.Address),
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	view := handler.ToOrderView(*order)
	// Return the guest access token exactly once, at creation time.
	view.AccessToken = order.AccessToken
	response.Created(c, view)
}

func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.Orders.Get(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*order))
}

func (h *Handler) CancelOrder(c *gin.Context) {
	order, err := h.Orders.Cancel(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "order.cancel", "order", order.ID, nil)
	response.OK(c, handler.ToOrderView(*order))
}

// ConfirmReceipt lets a customer confirm delivery of their own shipped order.
func (h *Handler) ConfirmReceipt(c *gin.Context) {
	if _, err := h.Orders.Get(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	order, err := h.Orders.MarkCompleted(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*order))
}

// --- Guest order endpoints (authorised by an unguessable access token) ---

func (h *Handler) GuestOrder(c *gin.Context) {
	order, err := h.Orders.FindByAccessToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*order))
}

func (h *Handler) GuestPay(c *gin.Context) {
	order, err := h.Orders.FindByAccessToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req createPaymentRequest
	_ = c.ShouldBindJSON(&req)
	res, err := h.Payments.Create(c.Request.Context(), service.CreatePaymentInput{
		UserID: "", OrderID: order.ID, ProviderName: req.Provider, ReturnURL: req.ReturnURL,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, paymentView{
		ID: res.Payment.ID, OrderID: res.Payment.OrderID, Provider: res.Payment.Provider,
		Status: string(res.Payment.Status), AmountCents: res.Payment.AmountCents,
		Currency: res.Payment.Currency, RedirectURL: res.RedirectURL,
	})
}

func (h *Handler) GuestCancel(c *gin.Context) {
	order, err := h.Orders.FindByAccessToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	cancelled, err := h.Orders.Cancel(c.Request.Context(), "", order.ID, true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*cancelled))
}

func (h *Handler) GuestComplete(c *gin.Context) {
	order, err := h.Orders.FindByAccessToken(c.Request.Context(), c.Param("token"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	completed, err := h.Orders.MarkCompleted(c.Request.Context(), order.ID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToOrderView(*completed))
}

// GuestSimulate confirms a sandbox payment for a guest order. The provider
// reference is unguessable, which authorises the call.
func (h *Handler) GuestSimulate(c *gin.Context) {
	var req simulatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	if err := h.Payments.Simulate(c.Request.Context(), "", req.Provider, req.ProviderRef); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"status": "succeeded"})
}
