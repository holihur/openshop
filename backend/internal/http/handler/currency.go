package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type exchangeRateView struct {
	Currency  string `json:"currency"`
	RateMicro int64  `json:"rateMicro"`
	UpdatedAt string `json:"updatedAt"`
}

type setCurrencyRateRequest struct {
	RateMicro int64 `json:"rateMicro" binding:"required,gt=0"`
}

// ListCurrencies is public so the storefront can offer a currency switcher.
func (h *Handler) ListCurrencies(c *gin.Context) {
	rates, err := h.Currency.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	views := make([]exchangeRateView, 0, len(rates))
	for _, r := range rates {
		views = append(views, toExchangeRateView(r))
	}
	response.OK(c, gin.H{"base": h.Currency.Base(), "rates": views})
}

func (h *Handler) SetCurrencyRate(c *gin.Context) {
	var req setCurrencyRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, wrapBind(err))
		return
	}
	rate, err := h.Currency.SetRate(c.Request.Context(), c.Param("code"), req.RateMicro)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toExchangeRateView(*rate))
}

func toExchangeRateView(r domain.ExchangeRate) exchangeRateView {
	return exchangeRateView{
		Currency:  r.Currency,
		RateMicro: r.RateMicro,
		UpdatedAt: r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
