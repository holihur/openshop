package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type ExchangeRateView struct {
	Currency  string `json:"currency"`
	RateMicro int64  `json:"rateMicro"`
	UpdatedAt string `json:"updatedAt"`
}

// ListCurrencies is shared: the storefront offers a currency switcher and the
// ops console shows the current rates.
func (h *Handler) ListCurrencies(c *gin.Context) {
	rates, err := h.Currency.List(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	views := make([]ExchangeRateView, 0, len(rates))
	for _, r := range rates {
		views = append(views, ToExchangeRateView(r))
	}
	response.OK(c, gin.H{"base": h.Currency.Base(), "rates": views})
}

func ToExchangeRateView(r domain.ExchangeRate) ExchangeRateView {
	return ExchangeRateView{
		Currency:  r.Currency,
		RateMicro: r.RateMicro,
		UpdatedAt: r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
