package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CustomerService backs the ops customer directory: listing, lookup and
// enable/disable.
type CustomerService struct {
	users port.UserRepository
	clock port.Clock
}

func NewCustomerService(users port.UserRepository, clock port.Clock) *CustomerService {
	return &CustomerService{users: users, clock: clock}
}

func (s *CustomerService) List(ctx context.Context, f domain.UserFilter) (domain.Page[domain.User], error) {
	return s.users.List(ctx, f)
}

func (s *CustomerService) Get(ctx context.Context, id string) (*domain.User, error) {
	return s.users.FindByID(ctx, id)
}

type UpdateCustomerInput struct {
	Name   *string
	Status *domain.UserStatus
}

func (s *CustomerService) Update(ctx context.Context, id string, in UpdateCustomerInput) (*domain.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		u.Name = strings.TrimSpace(*in.Name)
	}
	if in.Status != nil {
		switch *in.Status {
		case domain.UserActive, domain.UserDisabled:
			u.Status = *in.Status
		default:
			return nil, fmt.Errorf("%w: invalid status", domain.ErrInvalidArgument)
		}
	}
	u.UpdatedAt = s.clock.Now()
	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
