package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type OrderItemView struct {
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

type OrderView struct {
	ID                  string          `json:"id"`
	OrderNo             string          `json:"orderNo"`
	Status              string          `json:"status"`
	Currency            string          `json:"currency"`
	SubtotalCents       int64           `json:"subtotalCents"`
	DiscountCents       int64           `json:"discountCents"`
	WalletCents         int64           `json:"walletCents"`
	PointsUsed          int64           `json:"pointsUsed"`
	PointsDiscountCents int64           `json:"pointsDiscountCents"`
	CouponCode          string          `json:"couponCode,omitempty"`
	ShippingCents       int64           `json:"shippingCents"`
	TaxCents            int64           `json:"taxCents"`
	ShippingMethod      string          `json:"shippingMethod,omitempty"`
	TotalCents          int64           `json:"totalCents"`
	RefundedCents       int64           `json:"refundedCents"`
	Items               []OrderItemView `json:"items"`
	PaymentID           string          `json:"paymentId"`
	ShippingAddress     *AddressView    `json:"shippingAddress,omitempty"`
	TrackingNo          string          `json:"trackingNo,omitempty"`
	ShippedAt           string          `json:"shippedAt,omitempty"`
	CompletedAt         string          `json:"completedAt,omitempty"`
	ExpiresAt           string          `json:"expiresAt"`
	PaidAt              string          `json:"paidAt,omitempty"`
	CreatedAt           string          `json:"createdAt"`
	// AccessToken is returned once for guest orders so the buyer can view and
	// pay without an account.
	AccessToken string `json:"accessToken,omitempty"`
}

type AddressView struct {
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

// ListOrders is shared by the storefront (own orders) and ops (all orders).
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
	filter.Page, filter.PageSize = ParsePage(c, 10)

	page, err := h.Orders.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]OrderView, 0, len(page.Items))
	for _, o := range page.Items {
		out = append(out, ToOrderView(o))
	}
	response.Paginated(c, out, page.Total, page.Page, page.PageSize)
}

// DownloadInvoice streams the order as a PDF. Owners and admins may fetch it.
func (h *Handler) DownloadInvoice(c *gin.Context) {
	pdf, err := h.Orders.Invoice(c.Request.Context(), c.Param("id"), middleware.UserID(c), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="invoice.pdf"`)
	c.Data(200, "application/pdf", pdf)
}

func ToOrderView(o domain.Order) OrderView {
	items := make([]OrderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, OrderItemView{
			ID: it.ID, ProductID: it.ProductID, VariantID: it.VariantID, VariantName: it.VariantName,
			SKU: it.SKU, Title: it.Title, PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	view := OrderView{
		ID: o.ID, OrderNo: o.OrderNo, Status: string(o.Status), Currency: o.Currency,
		SubtotalCents: o.SubtotalCents, DiscountCents: o.DiscountCents, CouponCode: o.CouponCode,
		WalletCents: o.WalletCents, PointsUsed: o.PointsUsed, PointsDiscountCents: o.PointsDiscountCents,
		ShippingCents: o.ShippingCents, TaxCents: o.TaxCents, ShippingMethod: o.ShippingMethodName,
		TotalCents: o.TotalCents, Items: items, PaymentID: o.PaymentID,
		RefundedCents: o.RefundedCents,
		TrackingNo:    o.TrackingNo,
		ExpiresAt:     o.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:     o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	// The guest access token is only attached by Checkout, never on reads.
	if o.ShippingAddress != nil {
		view.ShippingAddress = &AddressView{
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
