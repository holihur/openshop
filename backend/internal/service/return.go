package service

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ReturnService handles customer return requests (RMA): a customer asks to send
// a delivered order back and ops approves or rejects it. Approved returns are
// refunded through the normal refund path (with restock).
type ReturnService struct {
	returns port.ReturnRepository
	orders  port.OrderRepository
	ids     port.IDGenerator
	clock   port.Clock
}

func NewReturnService(returns port.ReturnRepository, orders port.OrderRepository, ids port.IDGenerator, clock port.Clock) *ReturnService {
	return &ReturnService{returns: returns, orders: orders, ids: ids, clock: clock}
}

// Request creates a return request for a delivered order owned by the user.
func (s *ReturnService) Request(ctx context.Context, userID, orderID, reason string) (*domain.ReturnRequest, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrNotFound
	}
	if order.Status != domain.OrderCompleted {
		return nil, domain.ErrInvalidArgument
	}

	existing, err := s.returns.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	for _, r := range existing {
		if r.Status == domain.ReturnRequested || r.Status == domain.ReturnApproved {
			return nil, domain.ErrConflict
		}
	}

	now := s.clock.Now()
	req := &domain.ReturnRequest{
		ID: s.ids.NewID(), OrderID: orderID, UserID: userID, Reason: reason,
		Status: domain.ReturnRequested, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.returns.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// ListByOrder returns the return requests for an order to its owner or an admin.
func (s *ReturnService) ListByOrder(ctx context.Context, requesterID, orderID string, admin bool) ([]domain.ReturnRequest, error) {
	order, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !admin && order.UserID != requesterID {
		return nil, domain.ErrNotFound
	}
	return s.returns.ListByOrder(ctx, orderID)
}

// List returns return requests for ops moderation.
func (s *ReturnService) List(ctx context.Context, f domain.ReturnFilter) (domain.Page[domain.ReturnRequest], error) {
	return s.returns.List(ctx, f)
}

func (s *ReturnService) Approve(ctx context.Context, id string) (*domain.ReturnRequest, error) {
	return s.setStatus(ctx, id, domain.ReturnApproved)
}

func (s *ReturnService) Reject(ctx context.Context, id string) (*domain.ReturnRequest, error) {
	return s.setStatus(ctx, id, domain.ReturnRejected)
}

func (s *ReturnService) setStatus(ctx context.Context, id string, status domain.ReturnStatus) (*domain.ReturnRequest, error) {
	req, err := s.returns.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Status != domain.ReturnRequested {
		return nil, domain.ErrConflict
	}
	req.Status = status
	req.UpdatedAt = s.clock.Now()
	if err := s.returns.Update(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}
