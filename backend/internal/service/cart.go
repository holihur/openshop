package service

import (
	"context"
	"fmt"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// CartService manages the per-user cart held in the shared cart repository.
type CartService struct {
	carts    port.CartRepository
	products port.ProductRepository
}

func NewCartService(carts port.CartRepository, products port.ProductRepository) *CartService {
	return &CartService{carts: carts, products: products}
}

func (s *CartService) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	return s.carts.Get(ctx, userID)
}

func (s *CartService) AddItem(ctx context.Context, userID, productID string, quantity int) (*domain.Cart, error) {
	if quantity < 1 {
		quantity = 1
	}
	p, err := s.products.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProductPublished {
		return nil, fmt.Errorf("%w: product is not available", domain.ErrInvalidArgument)
	}

	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	existing := 0
	for _, it := range cart.Items {
		if it.ProductID == productID {
			existing = it.Quantity
			break
		}
	}
	if p.Stock < existing+quantity {
		return nil, domain.ErrInsufficientStock
	}

	cart.AddItem(domain.CartItem{
		ProductID:  p.ID,
		Title:      p.Title,
		CoverImage: p.CoverImage,
		PriceCents: p.PriceCents,
		Currency:   p.Currency,
		Quantity:   quantity,
	})
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) SetQuantity(ctx context.Context, userID, productID string, quantity int) (*domain.Cart, error) {
	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quantity > 0 {
		p, err := s.products.FindByID(ctx, productID)
		if err != nil {
			return nil, err
		}
		if p.Stock < quantity {
			return nil, domain.ErrInsufficientStock
		}
	}
	cart.SetQuantity(productID, quantity)
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, userID, productID string) (*domain.Cart, error) {
	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	cart.RemoveItem(productID)
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) Clear(ctx context.Context, userID string) error {
	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return err
	}
	cart.Clear()
	return s.carts.Save(ctx, cart)
}
