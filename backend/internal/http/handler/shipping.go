package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type ShippingMethodView struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	FlatRateCents      int64  `json:"flatRateCents"`
	FreeThresholdCents int64  `json:"freeThresholdCents"`
	MinDays            int    `json:"minDays"`
	MaxDays            int    `json:"maxDays"`
	Active             bool   `json:"active"`
	Sort               int    `json:"sort"`
}

// ListShippingMethods is shared: the storefront needs the options and rates,
// the ops console manages them.
func (h *Handler) ListShippingMethods(c *gin.Context) {
	methods, err := h.Shipping.List(c.Request.Context(), true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, ToShippingViews(methods))
}

// DeliveryEstimates reports the cost and delivery window of every active
// shipping method for the current cart, so the storefront can show the
// expectation before the shopper reaches checkout.
func (h *Handler) DeliveryEstimates(c *gin.Context) {
	estimates, err := h.Shipping.Estimate(
		c.Request.Context(),
		int64(queryInt(c, "subtotalCents", 0)),
		int64(queryInt(c, "weightGrams", 0)),
		c.Query("province"),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, estimates)
}

func ToShippingView(m domain.ShippingMethod) ShippingMethodView {
	return ShippingMethodView{
		ID: m.ID, Code: m.Code, Name: m.Name, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, MinDays: m.MinDays, MaxDays: m.MaxDays,
		Active: m.Active, Sort: m.Sort,
	}
}

func ToShippingViews(methods []domain.ShippingMethod) []ShippingMethodView {
	out := make([]ShippingMethodView, 0, len(methods))
	for _, m := range methods {
		out = append(out, ToShippingView(m))
	}
	return out
}
