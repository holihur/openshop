package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type orderItemView struct {
	ID         string `json:"id"`
	ProductID  string `json:"productId"`
	Title      string `json:"title"`
	PriceCents int64  `json:"priceCents"`
	Quantity   int    `json:"quantity"`
	Subtotal   int64  `json:"subtotal"`
}

type orderView struct {
	ID         string          `json:"id"`
	OrderNo    string          `json:"orderNo"`
	Status     string          `json:"status"`
	Currency   string          `json:"currency"`
	TotalCents int64           `json:"totalCents"`
	Items      []orderItemView `json:"items"`
	PaymentID  string          `json:"paymentId"`
	ExpiresAt  string          `json:"expiresAt"`
	PaidAt     string          `json:"paidAt,omitempty"`
	CreatedAt  string          `json:"createdAt"`
}

func (h *Handler) Checkout(c *gin.Context) {
	order, err := h.Orders.Checkout(c.Request.Context(), middleware.UserID(c))
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

func toOrderView(o domain.Order) orderView {
	items := make([]orderItemView, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemView{
			ID: it.ID, ProductID: it.ProductID, Title: it.Title,
			PriceCents: it.PriceCents, Quantity: it.Quantity, Subtotal: it.Subtotal,
		})
	}
	view := orderView{
		ID: o.ID, OrderNo: o.OrderNo, Status: string(o.Status), Currency: o.Currency,
		TotalCents: o.TotalCents, Items: items, PaymentID: o.PaymentID,
		ExpiresAt: o.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt: o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if o.PaidAt != nil {
		view.PaidAt = o.PaidAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return view
}
