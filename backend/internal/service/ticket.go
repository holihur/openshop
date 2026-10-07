package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CreateTicketInput is a new support ticket. A registered customer supplies only
// UserID; a guest supplies Email (and optionally Name).
type CreateTicketInput struct {
	UserID    string
	Email     string
	Name      string
	Subject   string
	Body      string
	Kind      domain.TicketKind
	OrderID   string
	ProductID string
}

// TicketPatch is a staff edit of the ticket metadata.
type TicketPatch struct {
	Status     *domain.TicketStatus
	Priority   *domain.TicketPriority
	Kind       *domain.TicketKind
	AssigneeID *string
}

// TicketService handles customer support threads (pre-sale and post-sale). A
// ticket is a threaded conversation; staff replies may be internal notes that
// the customer never sees.
type TicketService struct {
	tickets  port.TicketRepository
	users    port.UserRepository
	orders   port.OrderRepository
	ids      port.IDGenerator
	clock    port.Clock
	outbox   port.Outbox
	settings *SettingsService
}

func NewTicketService(
	tickets port.TicketRepository,
	users port.UserRepository,
	orders port.OrderRepository,
	ids port.IDGenerator,
	clock port.Clock,
	outbox port.Outbox,
	settings *SettingsService,
) *TicketService {
	return &TicketService{tickets: tickets, users: users, orders: orders, ids: ids, clock: clock, outbox: outbox, settings: settings}
}

// Create opens a ticket and stores the first message.
func (s *TicketService) Create(ctx context.Context, in CreateTicketInput) (*domain.Ticket, error) {
	subject := strings.TrimSpace(in.Subject)
	body := strings.TrimSpace(in.Body)
	if subject == "" || body == "" {
		return nil, domain.ErrInvalidArgument
	}
	if len(subject) > 300 {
		subject = subject[:300]
	}
	kind := in.Kind
	if !kind.Valid() {
		kind = domain.TicketOther
	}

	email := strings.TrimSpace(in.Email)
	name := strings.TrimSpace(in.Name)
	if in.UserID != "" {
		if user, err := s.users.FindByID(ctx, in.UserID); err == nil {
			email, name = user.Email, user.Name
		}
	}
	if email == "" {
		return nil, domain.ErrInvalidArgument
	}
	// A post-sale ticket may reference one of the caller's orders.
	if in.OrderID != "" {
		order, err := s.orders.FindByID(ctx, in.OrderID)
		if err != nil {
			return nil, err
		}
		if in.UserID == "" || order.UserID != in.UserID {
			return nil, domain.ErrNotFound
		}
	}

	now := s.clock.Now()
	ticket := &domain.Ticket{
		ID: s.ids.NewID(), UserID: in.UserID, Email: email, Name: name, Subject: subject,
		Kind: kind, Priority: domain.TicketNormal, Status: domain.TicketOpen,
		OrderID: in.OrderID, ProductID: in.ProductID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.tickets.Create(ctx, ticket); err != nil {
		return nil, err
	}
	role := "customer"
	if in.UserID == "" {
		role = "customer"
	}
	msg := &domain.TicketMessage{
		ID: s.ids.NewID(), TicketID: ticket.ID, AuthorID: in.UserID, AuthorRole: role,
		Body: body, CreatedAt: now,
	}
	if err := s.tickets.AddMessage(ctx, msg); err != nil {
		return nil, err
	}
	s.notify(ctx, ticket, msg)
	return ticket, nil
}

// List returns tickets for the support queue.
func (s *TicketService) List(ctx context.Context, f domain.TicketFilter) (domain.Page[domain.Ticket], error) {
	return s.tickets.List(ctx, f)
}

// Get returns a ticket and its thread. A customer only ever sees their own
// tickets and never sees internal notes.
func (s *TicketService) Get(ctx context.Context, id, requesterID string, staff bool) (*domain.Ticket, []domain.TicketMessage, error) {
	ticket, err := s.tickets.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !staff && ticket.UserID != requesterID {
		return nil, nil, domain.ErrNotFound
	}
	messages, err := s.tickets.ListMessages(ctx, id, staff)
	if err != nil {
		return nil, nil, err
	}
	s.attachAuthors(ctx, messages)
	return ticket, messages, nil
}

// Reply appends a message to the thread. Only staff may post internal notes.
func (s *TicketService) Reply(ctx context.Context, id, authorID, role, body string, internal bool) (*domain.TicketMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrInvalidArgument
	}
	ticket, err := s.tickets.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	staff := role == "staff" || role == "system"
	if !staff && ticket.UserID != authorID {
		return nil, domain.ErrNotFound
	}
	if !staff && ticket.Status.Closed() {
		return nil, domain.ErrConflict
	}
	if internal && !staff {
		return nil, domain.ErrForbidden
	}

	now := s.clock.Now()
	msg := &domain.TicketMessage{
		ID: s.ids.NewID(), TicketID: id, AuthorID: authorID, AuthorRole: role,
		Body: body, Internal: internal, CreatedAt: now,
	}
	if err := s.tickets.AddMessage(ctx, msg); err != nil {
		return nil, err
	}

	// A public staff reply moves the ticket to "pending" (waiting on the
	// customer); a customer reply reopens it.
	next := ticket.Status
	if internal {
		// internal notes never change the status
	} else if staff {
		if next == domain.TicketOpen {
			next = domain.TicketPending
		}
	} else {
		next = domain.TicketOpen
	}
	if next != ticket.Status {
		ticket.Status = next
		ticket.UpdatedAt = now
		if err := s.tickets.Update(ctx, ticket); err != nil {
			return nil, err
		}
	}
	if !internal {
		s.notify(ctx, ticket, msg)
	}
	return msg, nil
}

// Update applies a staff edit (status, priority, kind, assignee).
func (s *TicketService) Update(ctx context.Context, id string, patch TicketPatch) (*domain.Ticket, error) {
	ticket, err := s.tickets.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Status != nil {
		if !patch.Status.Valid() {
			return nil, domain.ErrInvalidArgument
		}
		ticket.Status = *patch.Status
	}
	if patch.Priority != nil {
		if !patch.Priority.Valid() {
			return nil, domain.ErrInvalidArgument
		}
		ticket.Priority = *patch.Priority
	}
	if patch.Kind != nil {
		if !patch.Kind.Valid() {
			return nil, domain.ErrInvalidArgument
		}
		ticket.Kind = *patch.Kind
	}
	if patch.AssigneeID != nil {
		ticket.AssigneeID = *patch.AssigneeID
	}
	ticket.UpdatedAt = s.clock.Now()
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

// CountOpen counts tickets that still need attention.
func (s *TicketService) CountOpen(ctx context.Context) (int64, error) {
	return s.tickets.CountOpen(ctx)
}

// attachAuthors fills in display names for staff so the console can show who
// wrote what without an extra round trip per message.
func (s *TicketService) attachAuthors(ctx context.Context, messages []domain.TicketMessage) {
	cache := map[string]string{}
	for i := range messages {
		if messages[i].AuthorRole != "staff" || messages[i].AuthorID == "" {
			continue
		}
		name, ok := cache[messages[i].AuthorID]
		if !ok {
			if user, err := s.users.FindByID(ctx, messages[i].AuthorID); err == nil {
				name = user.Name
			}
			cache[messages[i].AuthorID] = name
		}
		messages[i].AuthorName = name
	}
}

// notify writes a support event to the outbox so downstream consumers can send
// the customer an email. Failures are non-fatal: the ticket is already stored.
func (s *TicketService) notify(ctx context.Context, ticket *domain.Ticket, msg *domain.TicketMessage) {
	if s.outbox == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"ticketId": ticket.ID, "number": ticket.Number, "subject": ticket.Subject,
		"email": ticket.Email, "internal": msg.Internal, "messageId": msg.ID,
	})
	if err != nil {
		return
	}
	_ = s.outbox.Enqueue(ctx, port.Event{ID: s.ids.NewID(), Subject: "support.ticket.message", Payload: payload})
}
