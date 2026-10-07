package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/response"
)

// ListPaymentMethods returns the payment channels a shopper can choose at
// checkout. The set and order come from the ops console configuration.
func (h *Handler) ListPaymentMethods(c *gin.Context) {
	response.OK(c, h.Payments.Methods(c.Request.Context()))
}
