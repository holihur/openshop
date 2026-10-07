package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

// ExportAccount returns the caller's full data bundle (GDPR access request).
func (h *Handler) ExportAccount(c *gin.Context) {
	userID := middleware.UserID(c)
	export, err := h.Account.Export(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=openshop-export.json")
	response.OK(c, export)
}

// DeleteAccount erases the caller's personal data (GDPR erasure).
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID := middleware.UserID(c)
	if err := h.Account.Delete(c.Request.Context(), userID); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "account.delete", "user", userID, nil)
	response.NoContent(c)
}
