package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type addressRequest struct {
	Recipient  string `json:"recipient" binding:"required"`
	Phone      string `json:"phone"`
	Province   string `json:"province"`
	City       string `json:"city"`
	District   string `json:"district"`
	Line1      string `json:"line1" binding:"required"`
	PostalCode string `json:"postalCode"`
	Default    bool   `json:"default"`
}

func (h *Handler) ListAddresses(c *gin.Context) {
	addresses, err := h.Addresses.List(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]addressView, 0, len(addresses))
	for _, a := range addresses {
		out = append(out, addressView{
			ID: a.ID, Recipient: a.Recipient, Phone: a.Phone, Province: a.Province,
			City: a.City, District: a.District, Line1: a.Line1, PostalCode: a.PostalCode,
			Default: a.Default,
		})
	}
	response.OK(c, out)
}

func (h *Handler) CreateAddress(c *gin.Context) {
	var req addressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	a, err := h.Addresses.Create(c.Request.Context(), middleware.UserID(c), toAddressInput(req))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, addressView{
		ID: a.ID, Recipient: a.Recipient, Phone: a.Phone, Province: a.Province,
		City: a.City, District: a.District, Line1: a.Line1, PostalCode: a.PostalCode,
		Default: a.Default,
	})
}

func (h *Handler) UpdateAddress(c *gin.Context) {
	var req addressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	a, err := h.Addresses.Update(c.Request.Context(), middleware.UserID(c), c.Param("id"), toAddressInput(req))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, addressView{
		ID: a.ID, Recipient: a.Recipient, Phone: a.Phone, Province: a.Province,
		City: a.City, District: a.District, Line1: a.Line1, PostalCode: a.PostalCode,
		Default: a.Default,
	})
}

func (h *Handler) DeleteAddress(c *gin.Context) {
	if err := h.Addresses.Delete(c.Request.Context(), middleware.UserID(c), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) SetDefaultAddress(c *gin.Context) {
	a, err := h.Addresses.SetDefault(c.Request.Context(), middleware.UserID(c), c.Param("id"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, addressView{
		ID: a.ID, Recipient: a.Recipient, Phone: a.Phone, Province: a.Province,
		City: a.City, District: a.District, Line1: a.Line1, PostalCode: a.PostalCode,
		Default: a.Default,
	})
}

func toAddressInput(req addressRequest) service.AddressInput {
	return service.AddressInput{
		Recipient: req.Recipient, Phone: req.Phone, Province: req.Province,
		City: req.City, District: req.District, Line1: req.Line1,
		PostalCode: req.PostalCode, Default: req.Default,
	}
}
