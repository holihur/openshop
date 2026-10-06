package service

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ShippingService manages selectable shipping methods and zones.
type ShippingService struct {
	methods port.ShippingMethodRepository
	zones   port.ShippingZoneRepository
	ids     port.IDGenerator
	clock   port.Clock
}

func NewShippingService(methods port.ShippingMethodRepository, zones port.ShippingZoneRepository, ids port.IDGenerator, clock port.Clock) *ShippingService {
	return &ShippingService{methods: methods, zones: zones, ids: ids, clock: clock}
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

// --- Zones and per-zone rates ---

type ZoneInput struct {
	Name      string
	Provinces []string
	Active    bool
	Sort      int
}

func (s *ShippingService) ListZones(ctx context.Context, activeOnly bool) ([]domain.ShippingZone, error) {
	return s.zones.List(ctx, activeOnly)
}

func (s *ShippingService) CreateZone(ctx context.Context, in ZoneInput) (*domain.ShippingZone, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
	}
	now := s.clock.Now()
	z := &domain.ShippingZone{
		ID: s.ids.NewID(), Name: in.Name, Provinces: in.Provinces,
		Active: in.Active, Sort: in.Sort, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.zones.Create(ctx, z); err != nil {
		return nil, err
	}
	return z, nil
}

func (s *ShippingService) UpdateZone(ctx context.Context, id string, in ZoneInput) (*domain.ShippingZone, error) {
	z, err := s.zones.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	z.Name = in.Name
	z.Provinces = in.Provinces
	z.Active = in.Active
	z.Sort = in.Sort
	z.UpdatedAt = s.clock.Now()
	if err := s.zones.Update(ctx, z); err != nil {
		return nil, err
	}
	return z, nil
}

type RateInput struct {
	FlatRateCents      int64
	FreeThresholdCents int64
	PerKgCents         int64
}

func (s *ShippingService) SetRate(ctx context.Context, zoneID, methodID string, in RateInput) (*domain.ShippingRate, error) {
	if _, err := s.zones.FindByID(ctx, zoneID); err != nil {
		return nil, err
	}
	if _, err := s.methods.FindByID(ctx, methodID); err != nil {
		return nil, err
	}
	r := &domain.ShippingRate{
		ID: s.ids.NewID(), ZoneID: zoneID, MethodID: methodID,
		FlatRateCents: in.FlatRateCents, FreeThresholdCents: in.FreeThresholdCents,
		PerKgCents: in.PerKgCents,
	}
	if err := s.zones.UpsertRate(ctx, r); err != nil {
		return nil, err
	}
	return s.zones.FindRate(ctx, zoneID, methodID)
}
