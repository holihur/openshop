package front

import (
	"github.com/gin-gonic/gin"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/http/handler"
	"github.com/holihur/openshop/internal/http/middleware"
	"github.com/holihur/openshop/internal/http/response"
	"github.com/holihur/openshop/internal/service"
)

type createTicketRequest struct {
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Kind      string `json:"kind"`
	OrderID   string `json:"orderId"`
	ProductID string `json:"productId"`
	Email     string `json:"email"`
	Name      string `json:"name"`
}

type replyTicketRequest struct {
	Body string `json:"body"`
}

// CreateTicket opens a support ticket. Registered customers are identified by
// their session; guests may use the contact form with an email address.
func (h *Handler) CreateTicket(c *gin.Context) {
	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	userID := middleware.UserID(c)
	result, err := h.Tickets.Create(c.Request.Context(), service.CreateTicketInput{
		UserID: userID, Email: req.Email, Name: req.Name, Subject: req.Subject, Body: req.Body,
		Kind: domain.TicketKind(req.Kind), OrderID: req.OrderID, ProductID: req.ProductID,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToTicketView(*result))
}

// ListMyTickets lists the signed-in customer's tickets.
func (h *Handler) ListMyTickets(c *gin.Context) {
	page, size := handler.ParsePage(c, 20)
	filter := domain.TicketFilter{UserID: middleware.UserID(c), Page: page, PageSize: size}
	if s := c.Query("status"); s != "" {
		st := domain.TicketStatus(s)
		filter.Status = &st
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

// GetMyTicket returns a ticket and its thread to its owner.
func (h *Handler) GetMyTicket(c *gin.Context) {
	ticket, messages, err := h.Tickets.Get(c.Request.Context(), c.Param("id"), middleware.UserID(c), middleware.IsAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{
		"ticket":   handler.ToTicketView(*ticket),
		"messages": handler.ToTicketMessageViews(messages),
	})
}

// ReplyMyTicket appends a customer reply to the thread.
func (h *Handler) ReplyMyTicket(c *gin.Context) {
	var req replyTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, domain.ErrInvalidArgument)
		return
	}
	msg, err := h.Tickets.Reply(c.Request.Context(), c.Param("id"), middleware.UserID(c), "customer", req.Body, false)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, handler.ToTicketMessageView(*msg))
}
