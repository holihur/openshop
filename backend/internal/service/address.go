package service

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// AddressService manages a user's shipping address book.
type AddressService struct {
	addresses port.AddressRepository
	ids       port.IDGenerator
	clock     port.Clock
}

func NewAddressService(addresses port.AddressRepository, ids port.IDGenerator, clock port.Clock) *AddressService {
	return &AddressService{addresses: addresses, ids: ids, clock: clock}
}

type AddressInput struct {
	Recipient  string
	Phone      string
	Province   string
	City       string
	District   string
	Line1      string
	PostalCode string
	Default    bool
}

func (s *AddressService) List(ctx context.Context, userID string) ([]domain.Address, error) {
	return s.addresses.ListByUser(ctx, userID)
}

func (s *AddressService) Create(ctx context.Context, userID string, in AddressInput) (*domain.Address, error) {
	if in.Recipient == "" || in.Line1 == "" {
		return nil, fmt.Errorf("%w: recipient and address line are required", domain.ErrInvalidArgument)
	}
	existing, err := s.addresses.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	// The first address, or an explicit request, becomes the default.
	makeDefault := in.Default || len(existing) == 0
	if makeDefault {
		if err := s.addresses.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}

	now := s.clock.Now()
	a := &domain.Address{
		ID: s.ids.NewID(), UserID: userID, Recipient: in.Recipient, Phone: in.Phone,
		Province: in.Province, City: in.City, District: in.District, Line1: in.Line1,
		PostalCode: in.PostalCode, Default: makeDefault, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.addresses.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AddressService) Update(ctx context.Context, userID, id string, in AddressInput) (*domain.Address, error) {
	a, err := s.addresses.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, domain.ErrNotFound
	}
	if in.Recipient == "" || in.Line1 == "" {
		return nil, fmt.Errorf("%w: recipient and address line are required", domain.ErrInvalidArgument)
	}
	if in.Default && !a.Default {
		if err := s.addresses.ClearDefault(ctx, userID); err != nil {
			return nil, err
		}
	}
	a.Recipient, a.Phone = in.Recipient, in.Phone
	a.Province, a.City, a.District = in.Province, in.City, in.District
	a.Line1, a.PostalCode = in.Line1, in.PostalCode
	a.Default = in.Default
	a.UpdatedAt = s.clock.Now()
	if err := s.addresses.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AddressService) Delete(ctx context.Context, userID, id string) error {
	a, err := s.addresses.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if a.UserID != userID {
		return domain.ErrNotFound
	}
	return s.addresses.Delete(ctx, id)
}

func (s *AddressService) SetDefault(ctx context.Context, userID, id string) (*domain.Address, error) {
	a, err := s.addresses.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.UserID != userID {
		return nil, domain.ErrNotFound
	}
	if err := s.addresses.ClearDefault(ctx, userID); err != nil {
		return nil, err
	}
	a.Default = true
	a.UpdatedAt = s.clock.Now()
	if err := s.addresses.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}
