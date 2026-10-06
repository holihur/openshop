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
	h.audit(c, "shipping.create", "shipping_method", m.ID, nil)
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
	h.audit(c, "shipping.update", "shipping_method", m.ID, nil)
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

type shippingZoneView struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Provinces []string `json:"provinces"`
	Active    bool     `json:"active"`
	Sort      int      `json:"sort"`
}

type shippingZoneRequest struct {
	Name      string   `json:"name" binding:"required"`
	Provinces []string `json:"provinces"`
	Active    *bool    `json:"active"`
	Sort      int      `json:"sort"`
}

type shippingRateRequest struct {
	FlatRateCents      int64 `json:"flatRateCents" binding:"gte=0"`
	FreeThresholdCents int64 `json:"freeThresholdCents" binding:"gte=0"`
	PerKgCents         int64 `json:"perKgCents" binding:"gte=0"`
}

func (h *Handler) ListShippingZones(c *gin.Context) {
	zones, err := h.Shipping.ListZones(c.Request.Context(), false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]shippingZoneView, 0, len(zones))
	for _, z := range zones {
		out = append(out, toShippingZoneView(z))
	}
	response.OK(c, out)
}

func (h *Handler) CreateShippingZone(c *gin.Context) {
	var req shippingZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	z, err := h.Shipping.CreateZone(c.Request.Context(), service.ZoneInput{
		Name: req.Name, Provinces: req.Provinces, Active: active, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.audit(c, "shipping_zone.create", "shipping_zone", z.ID, nil)
	response.Created(c, toShippingZoneView(*z))
}

func (h *Handler) UpdateShippingZone(c *gin.Context) {
	var req shippingZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	z, err := h.Shipping.UpdateZone(c.Request.Context(), c.Param("id"), service.ZoneInput{
		Name: req.Name, Provinces: req.Provinces, Active: active, Sort: req.Sort,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.audit(c, "shipping_zone.update", "shipping_zone", z.ID, nil)
	response.OK(c, toShippingZoneView(*z))
}

func (h *Handler) SetShippingRate(c *gin.Context) {
	var req shippingRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	rate, err := h.Shipping.SetRate(c.Request.Context(), c.Param("zoneId"), c.Param("methodId"), service.RateInput{
		FlatRateCents: req.FlatRateCents, FreeThresholdCents: req.FreeThresholdCents, PerKgCents: req.PerKgCents,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.audit(c, "shipping_rate.set", "shipping_rate", rate.ID, map[string]string{"zoneId": rate.ZoneID, "methodId": rate.MethodID})
	response.OK(c, gin.H{
		"zoneId": rate.ZoneID, "methodId": rate.MethodID,
		"flatRateCents": rate.FlatRateCents, "freeThresholdCents": rate.FreeThresholdCents,
		"perKgCents": rate.PerKgCents,
	})
}

func toShippingZoneView(z domain.ShippingZone) shippingZoneView {
	return shippingZoneView{ID: z.ID, Name: z.Name, Provinces: z.Provinces, Active: z.Active, Sort: z.Sort}
}
