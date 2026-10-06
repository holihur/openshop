package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type shippingMethodView struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	FlatRateCents      int64  `json:"flatRateCents"`
	FreeThresholdCents int64  `json:"freeThresholdCents"`
	Active             bool   `json:"active"`
	Sort               int    `json:"sort"`
}

type shippingMethodRequest struct {
	Code               string `json:"code"`
	Name               string `json:"name" binding:"required"`
	FlatRateCents      int64  `json:"flatRateCents" binding:"gte=0"`
	FreeThresholdCents int64  `json:"freeThresholdCents" binding:"gte=0"`
	Active             *bool  `json:"active"`
	Sort               int    `json:"sort"`
}

// ListShippingMethods is public: the storefront needs the options and rates.
func (h *Handler) ListShippingMethods(c *gin.Context) {
	methods, err := h.Shipping.List(c.Request.Context(), true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toShippingViews(methods))
}

func (h *Handler) AdminListShippingMethods(c *gin.Context) {
	methods, err := h.Shipping.List(c.Request.Context(), false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toShippingViews(methods))
}

func (h *Handler) CreateShippingMethod(c *gin.Context) {
	var req shippingMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	m, err := h.Shipping.Create(c.Request.Context(), service.ShippingMethodInput{
		Code: req.Code, Name: req.Name, FlatRateCents: req.FlatRateCents,
		FreeThresholdCents: req.FreeThresholdCents, Active: active, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toShippingView(*m))
}

func (h *Handler) UpdateShippingMethod(c *gin.Context) {
	var req shippingMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	m, err := h.Shipping.Update(c.Request.Context(), c.Param("id"), service.ShippingMethodInput{
		Name: req.Name, FlatRateCents: req.FlatRateCents,
		FreeThresholdCents: req.FreeThresholdCents, Active: active, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toShippingView(*m))
}

func toShippingView(m domain.ShippingMethod) shippingMethodView {
	return shippingMethodView{
		ID: m.ID, Code: m.Code, Name: m.Name, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, Active: m.Active, Sort: m.Sort,
	}
}

func toShippingViews(methods []domain.ShippingMethod) []shippingMethodView {
	out := make([]shippingMethodView, 0, len(methods))
	for _, m := range methods {
		out = append(out, toShippingView(m))
	}
	return out
}
