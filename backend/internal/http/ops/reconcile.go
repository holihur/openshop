package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/response"
)

// ReconciliationReport returns the current money invariants for the console.
// It reads, so it writes no audit entry: watching the state is not an event.
func (h *Handler) ReconciliationReport(c *gin.Context) {
	report, err := h.Reconciliation.Latest(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"clean":       report.Clean(),
		"mismatches":  report.Mismatches(),
		"walletDrift": report.WalletDrift,
		"pointsDrift": report.PointsDrift,
		"orderDrift":  report.OrderDrift,
	})
}
