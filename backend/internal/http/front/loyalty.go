package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

// GetWallet returns the signed-in customer's stored-value balance.
func (h *Handler) GetWallet(c *gin.Context) {
	wallet, err := h.Wallet.Get(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToWalletView(*wallet))
}

// ListWalletTransactions lists the customer's wallet ledger.
func (h *Handler) ListWalletTransactions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Wallet.Transactions(c.Request.Context(), middleware.UserID(c), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.WalletTxView, 0, len(result.Items))
	for _, tx := range result.Items {
		out = append(out, handler.ToWalletTxView(tx))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

type topUpRequest struct {
	AmountCents int64 `json:"amountCents"`
}

// TopUpWallet credits the customer's wallet (sandbox/offline top-up).
func (h *Handler) TopUpWallet(c *gin.Context) {
	var req topUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	wallet, err := h.Wallet.TopUp(c.Request.Context(), middleware.UserID(c), req.AmountCents)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToWalletView(*wallet))
}

// GetPoints returns the signed-in customer's loyalty balance.
func (h *Handler) GetPoints(c *gin.Context) {
	account, err := h.Points.Get(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToPointsView(*account))
}

// ListPointsTransactions lists the customer's loyalty ledger.
func (h *Handler) ListPointsTransactions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Points.Transactions(c.Request.Context(), middleware.UserID(c), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.PointsTxView, 0, len(result.Items))
	for _, tx := range result.Items {
		out = append(out, handler.ToPointsTxView(tx))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// GetReferralSummary returns the customer's referral code and earnings.
func (h *Handler) GetReferralSummary(c *gin.Context) {
	summary, err := h.Commission.Summary(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToReferralSummaryView(*summary))
}

// ListMyCommissions lists the customer's own referral commissions.
func (h *Handler) ListMyCommissions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Commission.ListCommissions(c.Request.Context(), middleware.UserID(c), page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.CommissionView, 0, len(result.Items))
	for _, cm := range result.Items {
		out = append(out, handler.ToCommissionView(cm))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}
