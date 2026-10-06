package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type orderItemView struct {
	ID          string `json:"id"`
	ProductID   string `json:"productId"`
	VariantID   string `json:"variantId,omitempty"`
	VariantName string `json:"variantName,omitempty"`
	SKU         string `json:"sku,omitempty"`
	Title       string `json:"title"`
	PriceCents  int64  `json:"priceCents"`
	Quantity    int    `json:"quantity"`
	Subtotal    int64  `json:"subtotal"`
}

type orderView struct {
	ID              string          `json:"id"`
	OrderNo         string          `json:"orderNo"`
	Status          string          `json:"status"`
	Currency        string          `json:"currency"`
	SubtotalCents   int64           `json:"subtotalCents"`
	DiscountCents   int64           `json:"discountCents"`
	CouponCode      string          `json:"couponCode,omitempty"`
	ShippingCents   int64           `json:"shippingCents"`
	TaxCents        int64           `json:"taxCents"`
	ShippingMethod  string          `json:"shippingMethod,omitempty"`
	TotalCents      int64           `json:"totalCents"`
	Items           []orderItemView `json:"items"`
	PaymentID       string          `json:"paymentId"`
	ShippingAddress *addressView    `json:"shippingAddress,omitempty"`
	TrackingNo      string          `json:"trackingNo,omitempty"`
	ShippedAt       string          `json:"shippedAt,omitempty"`
	CompletedAt     string          `json:"completedAt,omitempty"`
	ExpiresAt       string          `json:"expiresAt"`
	PaidAt          string          `json:"paidAt,omitempty"`
	CreatedAt       string          `json:"createdAt"`
}

type addressView struct {
	ID         string `json:"id"`
	Recipient  string `json:"recipient"`
	Phone      string `json:"phone"`
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	Line1      string `json:"line1"`
	PostalCode string `json:"postalCode"`
	Default    bool   `json:"default"`
}

// CheckoutRequest is optional: callers may include a coupon, address and
// shipping method.
type CheckoutRequest struct {
	CouponCode       string `json:"couponCode"`
	AddressID        string `json:"addressId"`
	ShippingMethodID string `json:"shippingMethodId"`
}

func (h *Handler) Checkout(c *gin.Context) {
	var req CheckoutRequest
	// The body is optional, so binding failures are ignored.
	_ = c.ShouldBindJSON(&req)

	order, err := h.Orders.Checkout(c.Request.Context(), service.CheckoutInput{
		UserID:           middleware.UserID(c),
		CouponCode:       req.CouponCode,
		AddressID:        req.AddressID,
		ShippingMethodID: req.ShippingMethodID,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toOrderView(*order))
}

func (h *Handler) ListOrders(c *gin.Context) {
	filter := domain.OrderFilter{}
	if !middleware.IsAdmin(c) {
		filter.UserID = middleware.UserID(c)
	} else if uid := c.Query("userId"); uid != "" {
		filter.UserID = uid
	}
	if s := c.Query("status"); s != "" {
		st := domain.OrderStatus(s)
		filter.Status = &st
	}
	filter.Page, filter.PageSize = parsePage(c, 10)

	page, err := h.Orders.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]orderView, 0, len(page.Items))
	for _, o := range page.Items {
		out = append(out, toOrderView(o))
	}
	response.Paginated(c, out, page.Total, page.Page, page.PageSize)
}

func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.Orders.Get(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderView(*order))
}

func (h *Handler) CancelOrder(c *gin.Context) {
	order, err := h.Orders.Cancel(c.Request.Context(), middleware.UserID(c), c.Param("id"), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderView(*order))
}

type refundRequest struct {
	Reason string `json:"reason"`
}

// RefundOrder reverses a paid order through its payment provider (admin only).
func (h *Handler) RefundOrder(c *gin.Context) {
	var req refundRequest
	_ = c.ShouldBindJSON(&req)

	order, err := h.Payments.Refund(c.Request.Context(), c.Param("id"), req.Reason)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderView(*order))
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
	response.OK(c, toOrderView(*order))
}

// CompleteOrder closes a shipped order.
func (h *Handler) CompleteOrder(c *gin.Context) {
	order, err := h.Orders.MarkCompleted(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderView(*order))
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
	response.OK(c, toOrderView(*order))
}

func toOrderView(o domain.Order) orderView {
	items := make([]orderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemView{
			ID: it.ID, ProductID: it.ProductID, VariantID: it.VariantID, VariantName: it.VariantName,
			SKU: it.SKU, Title: it.Title, PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	view := orderView{
		ID: o.ID, OrderNo: o.OrderNo, Status: string(o.Status), Currency: o.Currency,
		SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, CouponCode: o.CouponCode,
		ShippingCents: o.ShippingCents, TaxCents: o.TaxCents, ShippingMethod: o.ShippingMethodName,
		TotalCents: o.TotalCents, Items: items, PaymentID: o.PaymentID,
		TrackingNo: o.TrackingNo,
		ExpiresAt:  o.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:  o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if o.ShippingAddress != nil {
		view.ShippingAddress = &addressView{
			ID: o.ShippingAddress.ID, Recipient: o.ShippingAddress.Recipient, Phone: o.ShippingAddress.Phone,
			Province: o.ShippingAddress.Province, City: o.ShippingAddress.City, District: o.ShippingAddress.District,
			Line1: o.ShippingAddress.Line1, PostalCode: o.ShippingAddress.PostalCode, Default: o.ShippingAddress.Default,
		}
	}
	if o.ShippedAt != nil {
		view.ShippedAt = o.ShippedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if o.CompletedAt != nil {
		view.CompletedAt = o.CompletedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if o.PaidAt != nil {
		view.PaidAt = o.PaidAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return view
}
