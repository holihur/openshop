package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
)

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

// ListAuditLogs returns the audit trail (admin only).
func (h *Handler) ListAuditLogs(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
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

// VerifyAuditChain recomputes the audit hash chain and reports the first entry
// that does not match, so tampering or deletion is detectable.
func (h *Handler) VerifyAuditChain(c *gin.Context) {
	checked, brokenID, err := h.Audit.VerifyChain(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"checked": checked, "intact": brokenID == "", "brokenId": brokenID})
}
