package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type createWithdrawalRequest struct {
	AmountCents int64  `json:"amountCents"`
	Method      string `json:"method"`
	AccountName string `json:"accountName"`
	AccountNo   string `json:"accountNo"`
	Note        string `json:"note"`
}

// RequestWithdrawal creates a withdrawal and holds the funds.
func (h *Handler) RequestWithdrawal(c *gin.Context) {
	var req createWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	result, err := h.Withdrawals.Request(c.Request.Context(), service.CreateWithdrawalInput{
		UserID: middleware.UserID(c), AmountCents: req.AmountCents, Method: req.Method,
		AccountName: req.AccountName, AccountNo: req.AccountNo, Note: req.Note,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToWithdrawalView(*result))
}

// ListMyWithdrawals lists the signed-in customer's requests.
func (h *Handler) ListMyWithdrawals(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Withdrawals.ListByUser(c.Request.Context(), middleware.UserID(c), page, size)
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

// CancelWithdrawal withdraws a pending request and returns the funds.
func (h *Handler) CancelWithdrawal(c *gin.Context) {
	result, err := h.Withdrawals.Cancel(c.Request.Context(), c.Param("id"), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, handler.ToWithdrawalView(*result))
}
