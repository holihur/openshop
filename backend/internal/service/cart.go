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
	variants port.VariantRepository
}

func NewCartService(carts port.CartRepository, products port.ProductRepository, variants port.VariantRepository) *CartService {
	return &CartService{carts: carts, products: products, variants: variants}
}

func (s *CartService) Get(ctx context.Context, userID string) (*domain.Cart, error) {
	return s.carts.Get(ctx, userID)
}

// line resolves the purchasable unit: either a specific variant or the product
// itself, returning the effective price, display name and available stock.
type line struct {
	priceCents  int64
	stock       int
	variantName string
	title       string
}

func (s *CartService) resolve(ctx context.Context, productID, variantID string) (*domain.Product, line, error) {
	p, err := s.products.FindByID(ctx, productID)
	if err != nil {
		return nil, line{}, err
	}
	if p.Status != domain.ProductPublished {
		return nil, line{}, fmt.Errorf("%w: product is not available", domain.ErrInvalidArgument)
	}

	l := line{priceCents: p.PriceCents, stock: p.Stock, title: p.Title}
	if variantID == "" {
		return p, l, nil
	}

	v, err := s.variants.FindByID(ctx, variantID)
	if err != nil {
		return nil, line{}, err
	}
	if v.ProductID != productID {
		return nil, line{}, fmt.Errorf("%w: variant does not belong to product", domain.ErrInvalidArgument)
	}
	if !v.Active {
		return nil, line{}, fmt.Errorf("%w: variant is not available", domain.ErrInvalidArgument)
	}
	l.priceCents = v.EffectivePrice(p.PriceCents)
	l.stock = v.Stock
	l.variantName = v.Name
	return p, l, nil
}

func (s *CartService) AddItem(ctx context.Context, userID, productID, variantID string, quantity int) (*domain.Cart, error) {
	if quantity < 1 {
		quantity = 1
	}
	p, l, err := s.resolve(ctx, productID, variantID)
	if err != nil {
		return nil, err
	}

	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	existing := 0
	for _, it := range cart.Items {
		if it.ProductID == productID && it.VariantID == variantID {
			existing = it.Quantity
			break
		}
	}
	if l.stock < existing+quantity {
		return nil, domain.ErrInsufficientStock
	}

	cart.AddItem(domain.CartItem{
		ProductID:   p.ID,
		VariantID:   variantID,
		VariantName: l.variantName,
		Title:       p.Title,
		CoverImage:  p.CoverImage,
		PriceCents:  l.priceCents,
		Currency:    p.Currency,
		Quantity:    quantity,
	})
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) SetQuantity(ctx context.Context, userID, productID, variantID string, quantity int) (*domain.Cart, error) {
	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quantity > 0 {
		_, l, err := s.resolve(ctx, productID, variantID)
		if err != nil {
			return nil, err
		}
		if l.stock < quantity {
			return nil, domain.ErrInsufficientStock
		}
	}
	cart.SetQuantity(productID, variantID, quantity)
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, userID, productID, variantID string) (*domain.Cart, error) {
	cart, err := s.carts.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	cart.RemoveItem(productID, variantID)
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
