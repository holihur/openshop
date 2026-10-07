package ops

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/response"
)

type broadcastNotificationRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Link  string `json:"link"`
}

// BroadcastNotification sends an in-app notification to every customer.
func (h *Handler) BroadcastNotification(c *gin.Context) {
	var req broadcastNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	count, err := h.Notifications.Broadcast(c.Request.Context(), req.Title, req.Body, req.Link)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "notification.broadcast", "notification", "", nil)
	response.OK(c, gin.H{"recipients": count})
}
