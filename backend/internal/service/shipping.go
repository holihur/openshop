package service

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ShippingService manages selectable shipping methods.
type ShippingService struct {
	methods port.ShippingMethodRepository
	ids     port.IDGenerator
	clock   port.Clock
}

func NewShippingService(methods port.ShippingMethodRepository, ids port.IDGenerator, clock port.Clock) *ShippingService {
	return &ShippingService{methods: methods, ids: ids, clock: clock}
}

type ShippingMethodInput struct {
	Code               string
	Name               string
	FlatRateCents      int64
	FreeThresholdCents int64
	Active             bool
	Sort               int
}

func (s *ShippingService) List(ctx context.Context, activeOnly bool) ([]domain.ShippingMethod, error) {
	return s.methods.List(ctx, activeOnly)
}

func (s *ShippingService) Create(ctx context.Context, in ShippingMethodInput) (*domain.ShippingMethod, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
	}
	if in.FlatRateCents < 0 || in.FreeThresholdCents < 0 {
		return nil, fmt.Errorf("%w: rates must be non-negative", domain.ErrInvalidArgument)
	}
	code := in.Code
	if code == "" {
		code = slugify(in.Name)
	}
	now := s.clock.Now()
	m := &domain.ShippingMethod{
		ID: s.ids.NewID(), Code: code, Name: in.Name,
		FlatRateCents: in.FlatRateCents, FreeThresholdCents: in.FreeThresholdCents,
		Active: in.Active, Sort: in.Sort, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.methods.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *ShippingService) Update(ctx context.Context, id string, in ShippingMethodInput) (*domain.ShippingMethod, error) {
	m, err := s.methods.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Name = in.Name
	m.FlatRateCents = in.FlatRateCents
	m.FreeThresholdCents = in.FreeThresholdCents
	m.Active = in.Active
	m.Sort = in.Sort
	m.UpdatedAt = s.clock.Now()
	if err := s.methods.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}
