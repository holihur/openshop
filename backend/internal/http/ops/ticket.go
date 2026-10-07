package ops

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

// ListTickets lists support tickets for the console queue.
func (h *Handler) ListTickets(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.TicketFilter{Page: page, PageSize: size, Search: c.Query("q")}
	if s := c.Query("status"); s != "" {
		st := domain.TicketStatus(s)
		filter.Status = &st
	}
	if k := c.Query("kind"); k != "" {
		kd := domain.TicketKind(k)
		filter.Kind = &kd
	}
	switch a := c.Query("assignee"); a {
	case "":
	case "me":
		id := middleware.UserID(c)
		filter.Assignee = &id
	case "unassigned":
		empty := ""
		filter.Assignee = &empty
	default:
		filter.Assignee = &a
	}
	result, err := h.Tickets.List(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]handler.TicketView, 0, len(result.Items))
	for _, t := range result.Items {
		out = append(out, handler.ToTicketView(t))
	}
	response.Paginated(c, out, result.Total, result.Page, result.PageSize)
}

// GetTicket returns a ticket and its full thread, including internal notes.
func (h *Handler) GetTicket(c *gin.Context) {
	ticket, messages, err := h.Tickets.Get(c.Request.Context(), c.Param("id"), middleware.UserID(c), true)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"ticket":   handler.ToTicketView(*ticket),
		"messages": handler.ToTicketMessageViews(messages),
	})
}

type replyTicketRequest struct {
	Body     string `json:"body"`
	Internal bool   `json:"internal"`
}

// ReplyTicket appends a staff reply (or an internal note) to the thread.
func (h *Handler) ReplyTicket(c *gin.Context) {
	var req replyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	msg, err := h.Tickets.Reply(c.Request.Context(), c.Param("id"), middleware.UserID(c), "staff", req.Body, req.Internal)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "ticket.reply", "ticket", c.Param("id"), map[string]string{"internal": strconv.FormatBool(req.Internal)})
	response.Created(c, handler.ToTicketMessageView(*msg))
}

type updateTicketRequest struct {
	Status     *string `json:"status"`
	Priority   *string `json:"priority"`
	Kind       *string `json:"kind"`
	AssigneeID *string `json:"assigneeId"`
}

// UpdateTicket applies a staff edit to the ticket metadata.
func (h *Handler) UpdateTicket(c *gin.Context) {
	var req updateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	patch := service.TicketPatch{}
	if req.Status != nil {
		s := domain.TicketStatus(*req.Status)
		patch.Status = &s
	}
	if req.Priority != nil {
		p := domain.TicketPriority(*req.Priority)
		patch.Priority = &p
	}
	if req.Kind != nil {
		k := domain.TicketKind(*req.Kind)
		patch.Kind = &k
	}
	patch.AssigneeID = req.AssigneeID

	ticket, err := h.Tickets.Update(c.Request.Context(), c.Param("id"), patch)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "ticket.update", "ticket", ticket.ID, nil)
	response.OK(c, handler.ToTicketView(*ticket))
}

// AssignTicketToMe assigns the ticket to the calling staff member.
func (h *Handler) AssignTicketToMe(c *gin.Context) {
	id := middleware.UserID(c)
	ticket, err := h.Tickets.Update(c.Request.Context(), c.Param("id"), service.TicketPatch{AssigneeID: &id})
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.RecordAudit(c, "ticket.assign", "ticket", ticket.ID, nil)
	response.OK(c, handler.ToTicketView(*ticket))
}
