package service

import (
	"context"
	"fmt"
	"time"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

const (
	categoryCacheKey = "catalog:categories"
	productCacheKey  = "catalog:product:"
	catalogTTL       = 10 * time.Minute
)

// CatalogService serves categories and products. Reads are cache-aside; because
// the cache is Redis, all replicas share it and invalidations propagate
// instantly.
type CatalogService struct {
	categories port.CategoryRepository
	products   port.ProductRepository
	cache      port.Cache
	ids        port.IDGenerator
	clock      port.Clock
	currency   string
}

func NewCatalogService(
	categories port.CategoryRepository,
	products port.ProductRepository,
	cache port.Cache,
	ids port.IDGenerator,
	clock port.Clock,
	currency string,
) *CatalogService {
	return &CatalogService{categories: categories, products: products, cache: cache, ids: ids, clock: clock, currency: currency}
}

type CreateCategoryInput struct {
	Name     string
	Slug     string
	ParentID string
	Sort     int
}

func (s *CatalogService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var cached []domain.Category
	if err := s.cache.GetJSON(ctx, categoryCacheKey, &cached); err == nil {
		return cached, nil
	}
	cats, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetJSON(ctx, categoryCacheKey, cats, catalogTTL)
	return cats, nil
}

func (s *CatalogService) CreateCategory(ctx context.Context, in CreateCategoryInput) (*domain.Category, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
	}
	slug := in.Slug
	if slug == "" {
		slug = slugify(in.Name)
	}
	now := s.clock.Now()
	c := &domain.Category{
		ID: s.ids.NewID(), Name: in.Name, Slug: slug, ParentID: in.ParentID,
		Sort: in.Sort, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.categories.Create(ctx, c); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, categoryCacheKey)
	return c, nil
}

type CreateProductInput struct {
	CategoryID  string
	Title       string
	Slug        string
	Description string
	PriceCents  int64
	Currency    string
	CoverImage  string
	Images      []string
	Status      domain.ProductStatus
	Stock       int
}

func (s *CatalogService) CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidArgument)
	}
	if in.PriceCents < 0 {
		return nil, fmt.Errorf("%w: price cannot be negative", domain.ErrInvalidArgument)
	}
	slug := in.Slug
	if slug == "" {
		slug = slugify(in.Title)
	}
	currency := in.Currency
	if currency == "" {
		currency = s.currency
	}
	status := in.Status
	if status == "" {
		status = domain.ProductDraft
	}
	now := s.clock.Now()
	p := &domain.Product{
		ID: s.ids.NewID(), CategoryID: in.CategoryID, Title: in.Title, Slug: slug,
		Description: in.Description, PriceCents: in.PriceCents, Currency: currency,
		CoverImage: in.CoverImage, Images: in.Images, Status: status, Stock: in.Stock,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.products.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

type UpdateProductInput struct {
	Title       *string
	Description *string
	PriceCents  *int64
	CoverImage  *string
	Images      []string
	Status      *domain.ProductStatus
	Stock       *int
	CategoryID  *string
}

func (s *CatalogService) UpdateProduct(ctx context.Context, id string, in UpdateProductInput) (*domain.Product, error) {
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.PriceCents != nil {
		p.PriceCents = *in.PriceCents
	}
	if in.CoverImage != nil {
		p.CoverImage = *in.CoverImage
	}
	if in.Images != nil {
		p.Images = in.Images
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.Stock != nil {
		p.Stock = *in.Stock
	}
	if in.CategoryID != nil {
		p.CategoryID = *in.CategoryID
	}
	p.UpdatedAt = s.clock.Now()
	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, productCacheKey+id)
	return p, nil
}

func (s *CatalogService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	var cached domain.Product
	if err := s.cache.GetJSON(ctx, productCacheKey+id, &cached); err == nil {
		return &cached, nil
	}
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetJSON(ctx, productCacheKey+id, p, catalogTTL)
	return p, nil
}

func (s *CatalogService) ListProducts(ctx context.Context, f domain.ProductFilter) (domain.Page[domain.Product], error) {
	f.Page, f.PageSize = clampPage(f.Page, f.PageSize, 20)
	return s.products.List(ctx, f)
}

// InvalidateProductCache is used after stock changes so the catalog reflects
// the new availability. It is exported for the order worker.
func (s *CatalogService) InvalidateProductCache(ctx context.Context, ids ...string) {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, productCacheKey+id)
	}
	if len(keys) > 0 {
		_ = s.cache.Delete(ctx, keys...)
	}
}
