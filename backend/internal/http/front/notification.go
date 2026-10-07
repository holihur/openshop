package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
)

// ListNotifications returns the signed-in customer's inbox.
func (h *Handler) ListNotifications(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	unreadOnly := c.Query("unread") == "true"
	result, err := h.Notifications.List(c.Request.Context(), middleware.UserID(c), unreadOnly, page, size)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Paginated(c, handler.ToNotificationViews(result.Items), result.Total, result.Page, result.PageSize)
}

// NotificationUnreadCount returns the unread badge count.
func (h *Handler) NotificationUnreadCount(c *gin.Context) {
	count, err := h.Notifications.UnreadCount(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

// MarkNotificationRead marks one notification read.
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	if err := h.Notifications.MarkRead(c.Request.Context(), c.Param("id"), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// MarkAllNotificationsRead clears the unread badge.
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	if err := h.Notifications.MarkAllRead(c.Request.Context(), middleware.UserID(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
