package ops

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

// ListWalletTransactions lists the wallet ledger across all customers.
func (h *Handler) ListWalletTransactions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.WalletTxFilter{Page: page, PageSize: size, UserID: c.Query("userId")}
	if tp := c.Query("type"); tp != "" {
		t := domain.WalletTransactionType(tp)
		filter.Type = &t
	}
	result, err := h.Wallet.ListTransactions(c.Request.Context(), filter)
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

type adjustWalletRequest struct {
	AmountCents int64  `json:"amountCents"`
	Description string `json:"description"`
}

// AdjustWallet applies an operator correction to a customer's balance.
func (h *Handler) AdjustWallet(c *gin.Context) {
	var req adjustWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	balance, err := h.Wallet.Adjust(c.Request.Context(), c.Param("id"), req.AmountCents, req.Description)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "wallet.adjust", "user", c.Param("id"), map[string]string{
		"amountCents": strconv.FormatInt(req.AmountCents, 10),
	})
	response.OK(c, gin.H{"balanceCents": balance})
}

type adjustPointsRequest struct {
	Points      int64  `json:"points"`
	Description string `json:"description"`
}

// AdjustPoints applies an operator correction to a customer's points.
func (h *Handler) AdjustPoints(c *gin.Context) {
	var req adjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	if err := h.Points.Adjust(c.Request.Context(), c.Param("id"), req.Points, req.Description); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "points.adjust", "user", c.Param("id"), map[string]string{
		"points": strconv.FormatInt(req.Points, 10),
	})
	response.OK(c, gin.H{"ok": true})
}

// ListCommissions lists referral commissions for the console.
func (h *Handler) ListCommissions(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.CommissionFilter{Page: page, PageSize: size, ReferrerID: c.Query("referrerId")}
	if s := c.Query("status"); s != "" {
		st := domain.CommissionStatus(s)
		filter.Status = &st
	}
	result, err := h.Commission.List(c.Request.Context(), filter)
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
