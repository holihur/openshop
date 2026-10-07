package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/service"
)

// RecordAudit records an action for the current actor. It is best-effort and
// never affects the response. It lives on the shared handler because both the
// storefront and ops surfaces audit actions.
func (h *Handler) RecordAudit(c *gin.Context, action, resourceType, resourceID string, meta map[string]string) {
	if h.Audit == nil {
		return
	}
	h.Audit.Record(c.Request.Context(), service.Entry{
		ActorID:      middleware.UserID(c),
		ActorRole:    middleware.Role(c),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     meta,
		IP:           c.ClientIP(),
	})
}
