package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

type setCurrencyRateRequest struct {
	RateMicro int64 `json:"rateMicro" binding:"required,gt=0"`
}

// SetCurrencyRate updates an exchange rate (admin only).
func (h *Handler) SetCurrencyRate(c *gin.Context) {
	var req setCurrencyRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	rate, err := h.Currency.SetRate(c.Request.Context(), c.Param("code"), req.RateMicro)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToExchangeRateView(*rate))
}
