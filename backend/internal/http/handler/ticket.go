package handler

import (
	"time"

	"github.com/holihur/openshop/internal/domain"
)

// TicketView is the API shape of a support ticket.
type TicketView struct {
	ID         string    `json:"id"`
	Number     int64     `json:"number"`
	Reference  string    `json:"reference"`
	UserID     string    `json:"userId,omitempty"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Subject    string    `json:"subject"`
	Kind       string    `json:"kind"`
	Priority   string    `json:"priority"`
	Status     string    `json:"status"`
	OrderID    string    `json:"orderId,omitempty"`
	ProductID  string    `json:"productId,omitempty"`
	AssigneeID string    `json:"assigneeId,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func ToTicketView(t domain.Ticket) TicketView {
	return TicketView{
		ID: t.ID, Number: t.Number, Reference: t.Reference(), UserID: t.UserID,
		Email: t.Email, Name: t.Name, Subject: t.Subject, Kind: string(t.Kind),
		Priority: string(t.Priority), Status: string(t.Status), OrderID: t.OrderID,
		ProductID: t.ProductID, AssigneeID: t.AssigneeID,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

// TicketMessageView is the API shape of one message in a ticket thread.
type TicketMessageView struct {
	ID         string    `json:"id"`
	AuthorID   string    `json:"authorId,omitempty"`
	AuthorRole string    `json:"authorRole"`
	AuthorName string    `json:"authorName,omitempty"`
	Body       string    `json:"body"`
	Internal   bool      `json:"internal"`
	CreatedAt  time.Time `json:"createdAt"`
}

func ToTicketMessageView(m domain.TicketMessage) TicketMessageView {
	return TicketMessageView{
		ID: m.ID, AuthorID: m.AuthorID, AuthorRole: m.AuthorRole, AuthorName: m.AuthorName,
		Body: m.Body, Internal: m.Internal, CreatedAt: m.CreatedAt,
	}
}

func ToTicketMessageViews(msgs []domain.TicketMessage) []TicketMessageView {
	out := make([]TicketMessageView, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, ToTicketMessageView(m))
	}
	return out
}
