package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type cartItemView struct {
	ProductID   string `json:"productId"`
	VariantID   string `json:"variantId,omitempty"`
	VariantName string `json:"variantName,omitempty"`
	Title       string `json:"title"`
	CoverImage  string `json:"coverImage"`
	PriceCents  int64  `json:"priceCents"`
	Currency    string `json:"currency"`
	Quantity    int    `json:"quantity"`
}

type cartView struct {
	Items      []cartItemView `json:"items"`
	TotalCents int64          `json:"totalCents"`
	TotalCount int            `json:"totalCount"`
}

type addCartItemRequest struct {
	ProductID string `json:"productId" binding:"required"`
	VariantID string `json:"variantId"`
	Quantity  int    `json:"quantity" binding:"gte=1"`
}

type updateCartItemRequest struct {
	VariantID string `json:"variantId"`
	Quantity  int    `json:"quantity"`
}

func (h *Handler) GetCart(c *gin.Context) {
	cart, err := h.Cart.Get(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCartView(cart))
}

func (h *Handler) AddCartItem(c *gin.Context) {
	var req addCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	cart, err := h.Cart.AddItem(c.Request.Context(), middleware.UserID(c), req.ProductID, req.VariantID, req.Quantity)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCartView(cart))
}

func (h *Handler) UpdateCartItem(c *gin.Context) {
	var req updateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	cart, err := h.Cart.SetQuantity(c.Request.Context(), middleware.UserID(c), c.Param("productId"), req.VariantID, req.Quantity)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCartView(cart))
}

func (h *Handler) RemoveCartItem(c *gin.Context) {
	cart, err := h.Cart.RemoveItem(c.Request.Context(), middleware.UserID(c), c.Param("productId"), c.Query("variantId"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCartView(cart))
}

func (h *Handler) ClearCart(c *gin.Context) {
	if err := h.Cart.Clear(c.Request.Context(), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

func toCartView(cart *domain.Cart) cartView {
	items := make([]cartItemView, 0, len(cart.Items))
	for _, it := range cart.Items {
		items = append(items, cartItemView{
			ProductID: it.ProductID, VariantID: it.VariantID, VariantName: it.VariantName,
			Title: it.Title, CoverImage: it.CoverImage,
			PriceCents: it.PriceCents, Currency: it.Currency, Quantity: it.Quantity,
		})
	}
	return cartView{Items: items, TotalCents: cart.TotalCents(), TotalCount: cart.TotalQuantity()}
}
