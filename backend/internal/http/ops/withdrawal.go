package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

// ListWithdrawals lists withdrawal requests for review.
func (h *Handler) ListWithdrawals(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.WithdrawalFilter{Page: page, PageSize: size, UserID: c.Query("userId")}
	if s := c.Query("status"); s != "" {
		st := domain.WithdrawalStatus(s)
		filter.Status = &st
	}
	result, err := h.Withdrawals.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.WithdrawalView, 0, len(result.Items))
	for _, w := range result.Items {
		out = append(out, handler.ToWithdrawalView(w))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// ApproveWithdrawal marks a request reviewed and awaiting offline payout.
func (h *Handler) ApproveWithdrawal(c *gin.Context) {
	w, err := h.Withdrawals.Approve(c.Request.Context(), c.Param("id"), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "withdrawal.approve", "withdrawal", w.ID, nil)
	response.OK(c, handler.ToWithdrawalView(*w))
}

type rejectWithdrawalRequest struct {
	Reason string `json:"reason"`
}

// RejectWithdrawal refuses a request and returns the held funds.
func (h *Handler) RejectWithdrawal(c *gin.Context) {
	var req rejectWithdrawalRequest
	_ = c.ShouldBindJSON(&req)
	w, err := h.Withdrawals.Reject(c.Request.Context(), c.Param("id"), middleware.UserID(c), req.Reason)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "withdrawal.reject", "withdrawal", w.ID, nil)
	response.OK(c, handler.ToWithdrawalView(*w))
}

type payWithdrawalRequest struct {
	Reference string `json:"reference"`
}

// PayWithdrawal records that the money was transferred offline.
func (h *Handler) PayWithdrawal(c *gin.Context) {
	var req payWithdrawalRequest
	_ = c.ShouldBindJSON(&req)
	w, err := h.Withdrawals.MarkPaid(c.Request.Context(), c.Param("id"), req.Reference)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "withdrawal.pay", "withdrawal", w.ID, map[string]string{"reference": w.PaidReference})
	response.OK(c, handler.ToWithdrawalView(*w))
}
