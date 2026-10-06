package service

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// WishlistService manages a user's saved products.
type WishlistService struct {
	wishlists port.WishlistRepository
	products  port.ProductRepository
}

func NewWishlistService(wishlists port.WishlistRepository, products port.ProductRepository) *WishlistService {
	return &WishlistService{wishlists: wishlists, products: products}
}

func (s *WishlistService) List(ctx context.Context, userID string) ([]domain.Product, error) {
	return s.wishlists.ListByUser(ctx, userID)
}

func (s *WishlistService) Add(ctx context.Context, userID, productID string) error {
	if _, err := s.products.FindByID(ctx, productID); err != nil {
		return err
	}
	return s.wishlists.Add(ctx, userID, productID)
}

func (s *WishlistService) Remove(ctx context.Context, userID, productID string) error {
	return s.wishlists.Remove(ctx, userID, productID)
}
