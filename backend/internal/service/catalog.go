package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	variants   port.VariantRepository
	faqs       port.ProductFAQRepository
	cache      port.Cache
	ids        port.IDGenerator
	clock      port.Clock
	currency   string
	settings   *SettingsService
}

func NewCatalogService(
	categories port.CategoryRepository,
	products port.ProductRepository,
	variants port.VariantRepository,
	faqs port.ProductFAQRepository,
	cache port.Cache,
	ids port.IDGenerator,
	clock port.Clock,
	currency string,
	settings *SettingsService,
) *CatalogService {
	return &CatalogService{categories: categories, products: products, variants: variants, faqs: faqs, cache: cache, ids: ids, clock: clock, currency: currency, settings: settings}
}

// searchSynonyms reads the operator-configured synonym map. Keys are matched
// case-insensitively against the whole search keyword, so a misconfigured entry
// can only ever add results, never break the query.
func (s *CatalogService) searchSynonyms(ctx context.Context) map[string][]string {
	if s.settings == nil {
		return nil
	}
	raw := strings.TrimSpace(s.settings.String(ctx, "search.synonyms"))
	if raw == "" {
		return nil
	}
	var configured map[string][]string
	if err := json.Unmarshal([]byte(raw), &configured); err != nil {
		return nil
	}
	out := make(map[string][]string, len(configured))
	for key, values := range configured {
		if key = strings.ToLower(strings.TrimSpace(key)); key != "" {
			out[key] = values
		}
	}
	return out
}

type CreateCategoryInput struct {
	Name     string
	Names    map[string]string
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
		ID: s.ids.NewID(), Name: in.Name, Names: cleanNames(in.Names), Slug: slug, ParentID: in.ParentID,
		Sort: in.Sort, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.categories.Create(ctx, c); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, categoryCacheKey)
	return c, nil
}

type UpdateCategoryInput struct {
	Name     *string
	Names    *map[string]string
	Slug     *string
	ParentID *string
	Sort     *int
}

func (s *CatalogService) UpdateCategory(ctx context.Context, id string, in UpdateCategoryInput) (*domain.Category, error) {
	cat, err := s.categories.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidArgument)
		}
		cat.Name = name
	}
	if in.Names != nil {
		cat.Names = cleanNames(*in.Names)
	}
	if in.Slug != nil && strings.TrimSpace(*in.Slug) != "" {
		cat.Slug = slugify(*in.Slug)
	}
	if in.ParentID != nil {
		cat.ParentID = *in.ParentID
	}
	if in.Sort != nil {
		cat.Sort = *in.Sort
	}
	cat.UpdatedAt = s.clock.Now()
	if err := s.categories.Update(ctx, cat); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, categoryCacheKey)
	return cat, nil
}

// cleanNames trims and drops empty localized names.
func cleanNames(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for locale, name := range in {
		locale = strings.ToLower(strings.TrimSpace(locale))
		name = strings.TrimSpace(name)
		if locale == "" || name == "" {
			continue
		}
		out[locale] = name
	}
	return out
}

type CreateProductInput struct {
	CategoryID  string
	Title       string
	Slug        string
	Names       map[string]string
	Description string
	PriceCents  int64
	CostCents   int64
	Currency    string
	CoverImage  string
	Images      []string
	Status      domain.ProductStatus
	Stock       int
	WeightGrams int
}

func (s *CatalogService) CreateProduct(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidArgument)
	}
	if in.PriceCents < 0 {
		return nil, fmt.Errorf("%w: price cannot be negative", domain.ErrInvalidArgument)
	}
	if in.CostCents < 0 {
		return nil, fmt.Errorf("%w: cost cannot be negative", domain.ErrInvalidArgument)
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
		Names:       cleanNames(in.Names),
		Description: in.Description, PriceCents: in.PriceCents, CostCents: in.CostCents, Currency: currency,
		CoverImage: in.CoverImage, Images: in.Images, Status: status, Stock: in.Stock,
		WeightGrams: in.WeightGrams,
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := s.products.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

type UpdateProductInput struct {
	Title       *string
	Names       map[string]string
	Description *string
	PriceCents  *int64
	CostCents   *int64
	CoverImage  *string
	Images      []string
	Status      *domain.ProductStatus
	Stock       *int
	CategoryID  *string
	WeightGrams *int
}

func (s *CatalogService) UpdateProduct(ctx context.Context, id string, in UpdateProductInput) (*domain.Product, error) {
	p, err := s.products.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Names != nil {
		p.Names = cleanNames(in.Names)
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.PriceCents != nil {
		p.PriceCents = *in.PriceCents
	}
	if in.CostCents != nil {
		p.CostCents = *in.CostCents
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
	if in.WeightGrams != nil {
		p.WeightGrams = *in.WeightGrams
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
	s.attachVariants(ctx, p)
	s.attachFAQs(ctx, p)
	_ = s.cache.SetJSON(ctx, productCacheKey+id, p, catalogTTL)
	return p, nil
}

// attachVariants loads a product's variants. Failure is non-fatal: the product
// still renders without its variant selector.
func (s *CatalogService) attachVariants(ctx context.Context, p *domain.Product) {
	if s.variants == nil {
		return
	}
	variants, err := s.variants.ListByProduct(ctx, p.ID)
	if err != nil {
		return
	}
	p.Variants = variants
}

// attachFAQs loads a product's FAQs. Failure is non-fatal.
func (s *CatalogService) attachFAQs(ctx context.Context, p *domain.Product) {
	if s.faqs == nil {
		return
	}
	faqs, err := s.faqs.ListByProduct(ctx, p.ID)
	if err != nil {
		return
	}
	p.FAQs = faqs
}

// FAQInput is a question/answer pair supplied by an operator.
type FAQInput struct {
	Question string
	Answer   string
}

// ListFAQs returns a product's FAQs.
func (s *CatalogService) ListFAQs(ctx context.Context, productID string) ([]domain.ProductFAQ, error) {
	return s.faqs.ListByProduct(ctx, productID)
}

// ReplaceFAQs swaps a product's FAQs for the given set. Blank pairs are dropped.
func (s *CatalogService) ReplaceFAQs(ctx context.Context, productID string, in []FAQInput) ([]domain.ProductFAQ, error) {
	now := s.clock.Now()
	faqs := make([]domain.ProductFAQ, 0, len(in))
	for i, f := range in {
		q := strings.TrimSpace(f.Question)
		a := strings.TrimSpace(f.Answer)
		if q == "" || a == "" {
			continue
		}
		faqs = append(faqs, domain.ProductFAQ{
			ID: s.ids.NewID(), ProductID: productID, Question: q, Answer: a, Sort: i,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	if err := s.faqs.Replace(ctx, productID, faqs); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, productCacheKey+productID)
	return faqs, nil
}

func (s *CatalogService) ListProducts(ctx context.Context, f domain.ProductFilter) (domain.Page[domain.Product], error) {
	f.Page, f.PageSize = clampPage(f.Page, f.PageSize, 20)
	if f.Keyword != "" {
		f.Synonyms = s.searchSynonyms(ctx)
	}
	page, err := s.products.List(ctx, f)
	if err != nil {
		return page, err
	}
	// Nothing matched exactly: try to guess what the shopper meant. A typo then
	// yields a suggestion instead of an empty page, without polluting the
	// results when the exact search already succeeded.
	if len(page.Items) == 0 && f.Keyword != "" && !f.Fuzzy {
		fuzzy := f
		fuzzy.Fuzzy = true
		fuzzy.Cursor, fuzzy.CursorMode = "", false
		if alt, err := s.products.List(ctx, fuzzy); err == nil && len(alt.Items) > 0 {
			alt.Fuzzy = true
			return alt, nil
		}
	}
	return page, nil
}

// Facets lists the available price range and attribute filters for a category.
func (s *CatalogService) Facets(ctx context.Context, categoryID string) (domain.ProductFacets, error) {
	return s.products.Facets(ctx, categoryID)
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

type CreateVariantInput struct {
	ProductID   string
	SKU         string
	Name        string
	PriceCents  int64
	CostCents   int64
	Stock       int
	WeightGrams int
	Attributes  map[string]string
	Sort        int
	Active      bool
}

func (s *CatalogService) CreateVariant(ctx context.Context, in CreateVariantInput) (*domain.Variant, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("%w: variant name is required", domain.ErrInvalidArgument)
	}
	if in.PriceCents < 0 || in.CostCents < 0 || in.Stock < 0 {
		return nil, fmt.Errorf("%w: price, cost and stock must be non-negative", domain.ErrInvalidArgument)
	}
	if _, err := s.products.FindByID(ctx, in.ProductID); err != nil {
		return nil, err
	}
	sku := in.SKU
	if sku == "" {
		sku = "SKU-" + strings.ToUpper(s.ids.NewID()[:8])
	}
	now := s.clock.Now()
	v := &domain.Variant{
		ID: s.ids.NewID(), ProductID: in.ProductID, SKU: sku, Name: in.Name,
		PriceCents: in.PriceCents, CostCents: in.CostCents, Stock: in.Stock, Attributes: in.Attributes,
		WeightGrams: in.WeightGrams,
		Sort:        in.Sort, Active: in.Active, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.variants.Create(ctx, v); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, productCacheKey+in.ProductID)
	return v, nil
}

type UpdateVariantInput struct {
	SKU         *string
	Name        *string
	PriceCents  *int64
	CostCents   *int64
	Stock       *int
	WeightGrams *int
	Attributes  map[string]string
	Sort        *int
	Active      *bool
}

func (s *CatalogService) UpdateVariant(ctx context.Context, id string, in UpdateVariantInput) (*domain.Variant, error) {
	v, err := s.variants.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.SKU != nil {
		v.SKU = *in.SKU
	}
	if in.Name != nil {
		v.Name = *in.Name
	}
	if in.PriceCents != nil {
		v.PriceCents = *in.PriceCents
	}
	if in.CostCents != nil {
		v.CostCents = *in.CostCents
	}
	if in.Stock != nil {
		v.Stock = *in.Stock
	}
	if in.WeightGrams != nil {
		v.WeightGrams = *in.WeightGrams
	}
	if in.Attributes != nil {
		v.Attributes = in.Attributes
	}
	if in.Sort != nil {
		v.Sort = *in.Sort
	}
	if in.Active != nil {
		v.Active = *in.Active
	}
	v.UpdatedAt = s.clock.Now()
	if err := s.variants.Update(ctx, v); err != nil {
		return nil, err
	}
	_ = s.cache.Delete(ctx, productCacheKey+v.ProductID)
	return v, nil
}

func (s *CatalogService) ListVariants(ctx context.Context, productID string) ([]domain.Variant, error) {
	return s.variants.ListByProduct(ctx, productID)
}
