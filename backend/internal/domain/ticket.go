package domain

import (
	"fmt"
	"time"
)

// TicketKind distinguishes a pre-sale question from a post-sale issue. It is
// used for routing and reporting, not for access control.
type TicketKind string

const (
	TicketPresale  TicketKind = "presale"
	TicketPostsale TicketKind = "postsale"
	TicketOther    TicketKind = "other"
)

// Valid reports whether the kind is one the store understands.
func (k TicketKind) Valid() bool {
	switch k {
	case TicketPresale, TicketPostsale, TicketOther:
		return true
	}
	return false
}

// TicketStatus is the lifecycle of a support ticket.
type TicketStatus string

const (
	TicketOpen     TicketStatus = "open"     // needs a staff reply
	TicketPending  TicketStatus = "pending"  // waiting on the customer
	TicketResolved TicketStatus = "resolved" // answered, awaiting confirmation
	TicketClosed   TicketStatus = "closed"   // finished, read-only
)

// Valid reports whether the status is known.
func (s TicketStatus) Valid() bool {
	switch s {
	case TicketOpen, TicketPending, TicketResolved, TicketClosed:
		return true
	}
	return false
}

// Closed reports whether a ticket no longer accepts customer replies.
func (s TicketStatus) Closed() bool { return s == TicketClosed }

// TicketPriority orders the support queue.
type TicketPriority string

const (
	TicketLow    TicketPriority = "low"
	TicketNormal TicketPriority = "normal"
	TicketHigh   TicketPriority = "high"
	TicketUrgent TicketPriority = "urgent"
)

// Valid reports whether the priority is known.
func (p TicketPriority) Valid() bool {
	switch p {
	case TicketLow, TicketNormal, TicketHigh, TicketUrgent:
		return true
	}
	return false
}

// Ticket is a customer support thread, optionally linked to an order
// (post-sale) or a product (pre-sale).
type Ticket struct {
	ID         string
	Number     int64
	UserID     string
	Email      string
	Name       string
	Subject    string
	Kind       TicketKind
	Priority   TicketPriority
	Status     TicketStatus
	OrderID    string
	ProductID  string
	AssigneeID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Reference is the human-friendly ticket id shown to customers ("T-1000").
func (t Ticket) Reference() string { return fmt.Sprintf("T-%d", t.Number) }

// TicketMessage is a single message in a ticket thread. Internal notes are only
// ever returned to staff.
type TicketMessage struct {
	ID         string
	TicketID   string
	AuthorID   string
	AuthorRole string // "customer" | "staff" | "system"
	AuthorName string // resolved for display; not persisted on the message
	Body       string
	Internal   bool
	CreatedAt  time.Time
}

// TicketFilter is a storage-agnostic query for the support queue. A non-empty
// UserID restricts the result to one customer (used by the storefront).
type TicketFilter struct {
	Status   *TicketStatus
	Kind     *TicketKind
	Assignee *string
	UserID   string
	Search   string
	Page     int
	PageSize int
}
