package ops

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/port"
)

type outboxView struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Status      string `json:"status"`
	Attempts    int    `json:"attempts"`
	LastError   string `json:"lastError,omitempty"`
	CreatedAt   string `json:"createdAt"`
	AvailableAt string `json:"availableAt"`
}

func toOutboxView(m port.OutboxMessage) outboxView {
	return outboxView{
		ID: m.ID, Subject: m.Subject, Status: m.Status, Attempts: m.Attempts,
		LastError: m.LastError, CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		AvailableAt: m.AvailableAt.UTC().Format(time.RFC3339),
	}
}

// ListOutbox shows the event queue so dead letters are visible instead of
// silently disappearing after the relay gives up on them.
func (h *Handler) ListOutbox(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	result, err := h.Outbox.List(c.Request.Context(), port.OutboxFilter{
		Status: c.Query("status"), Subject: c.Query("subject"), Page: page, PageSize: size,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]outboxView, 0, len(result.Items))
	for _, m := range result.Items {
		out = append(out, toOutboxView(m))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// OutboxStats reports queue depth, dead letters and the age of the oldest
// pending event.
func (h *Handler) OutboxStats(c *gin.Context) {
	stats, err := h.Outbox.Stats(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	body := gin.H{
		"pending": stats.Pending, "processing": stats.Processing,
		"published": stats.Published, "failed": stats.Failed,
	}
	if stats.OldestPending != nil {
		body["oldestPending"] = stats.OldestPending.UTC().Format(time.RFC3339)
		body["oldestPendingAgeSeconds"] = int64(time.Since(*stats.OldestPending).Seconds())
	}
	response.OK(c, body)
}

// ReplayOutbox returns a dead-lettered event to the queue.
func (h *Handler) ReplayOutbox(c *gin.Context) {
	if err := h.Outbox.Replay(c.Request.Context(), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "outbox.replay", "outbox", c.Param("id"), nil)
	response.OK(c, gin.H{"ok": true})
}
