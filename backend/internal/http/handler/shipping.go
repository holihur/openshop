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

func ToShippingView(m domain.ShippingMethod) ShippingMethodView {
	return ShippingMethodView{
		ID: m.ID, Code: m.Code, Name: m.Name, FlatRateCents: m.FlatRateCents,
		FreeThresholdCents: m.FreeThresholdCents, Active: m.Active, Sort: m.Sort,
	}
}

func ToShippingViews(methods []domain.ShippingMethod) []ShippingMethodView {
	out := make([]ShippingMethodView, 0, len(methods))
	for _, m := range methods {
		out = append(out, ToShippingView(m))
	}
	return out
}
