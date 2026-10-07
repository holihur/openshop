package front

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type createPaymentRequest struct {
	OrderID   string `json:"orderId" binding:"required"`
	Provider  string `json:"provider"`
	ReturnURL string `json:"returnUrl"`
}

type paymentView struct {
	ID          string `json:"id"`
	OrderID     string `json:"orderId"`
	Provider    string `json:"provider"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amountCents"`
	Currency    string `json:"currency"`
	RedirectURL string `json:"redirectUrl,omitempty"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	res, err := h.Payments.Create(c.Request.Context(), service.CreatePaymentInput{
		UserID: middleware.UserID(c), OrderID: req.OrderID,
		ProviderName: req.Provider, ReturnURL: req.ReturnURL,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, paymentView{
		ID: res.Payment.ID, OrderID: res.Payment.OrderID, Provider: res.Payment.Provider,
		Status: string(res.Payment.Status), AmountCents: res.Payment.AmountCents,
		Currency: res.Payment.Currency, RedirectURL: res.RedirectURL,
	})
}

type simulatePaymentRequest struct {
	ProviderRef string `json:"providerRef" binding:"required"`
	Provider    string `json:"provider"`
}

// SimulatePayment completes a sandbox payment. It only works with providers
// that implement port.SandboxProvider and only for the caller's own payment.
func (h *Handler) SimulatePayment(c *gin.Context) {
	var req simulatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, handler.WrapBind(err))
		return
	}
	if err := h.Payments.Simulate(c.Request.Context(), middleware.UserID(c), req.Provider, req.ProviderRef); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"status": "succeeded"})
}

// PaymentWebhook is deliberately unauthenticated: identity comes from the
// provider signature verified inside ParseWebhook. The endpoint is public and
// must stay idempotent.
func (h *Handler) PaymentWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		response.Fail(c, fmt.Errorf("read body: %w", err))
		return
	}
	headers := make(map[string]string, len(c.Request.Header))
	for k := range c.Request.Header {
		headers[k] = c.GetHeader(k)
	}
	if err := h.Payments.HandleWebhook(c.Request.Context(), c.Param("provider"), headers, body); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"received": true})
}
