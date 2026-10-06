package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

type addWishlistRequest struct {
	ProductID string `json:"productId" binding:"required"`
}

func (h *Handler) ListWishlist(c *gin.Context) {
	products, err := h.Wishlist.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toProductViews(products))
}

func (h *Handler) AddWishlist(c *gin.Context) {
	var req addWishlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	if err := h.Wishlist.Add(c.Request.Context(), middleware.UserID(c), req.ProductID); err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, gin.H{"saved": true})
}

func (h *Handler) RemoveWishlist(c *gin.Context) {
	if err := h.Wishlist.Remove(c.Request.Context(), middleware.UserID(c), c.Param("productId")); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}
