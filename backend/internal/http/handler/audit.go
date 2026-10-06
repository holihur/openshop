package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// audit records an action for the current actor. It is best-effort and never
// affects the response.
func (h *Handler) audit(c *gin.Context, action, resourceType, resourceID string, meta map[string]string) {
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

type auditLogView struct {
	ID           string            `json:"id"`
	ActorID      string            `json:"actorId"`
	ActorRole    string            `json:"actorRole"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IP           string            `json:"ip"`
	CreatedAt    string            `json:"createdAt"`
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	page, size := parsePage(c, 20)
	filter := domain.AuditFilter{
		ActorID:      c.Query("actorId"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resourceType"),
		Page:         page,
		PageSize:     size,
	}
	logs, err := h.Audit.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]auditLogView, 0, len(logs.Items))
	for _, l := range logs.Items {
		out = append(out, auditLogView{
			ID: l.ID, ActorID: l.ActorID, ActorRole: l.ActorRole, Action: l.Action,
			ResourceType: l.ResourceType, ResourceID: l.ResourceID, Metadata: l.Metadata,
			IP: l.IP, CreatedAt: l.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	response.Paginated(c, out, logs.Total, logs.Page, logs.PageSize)
}
