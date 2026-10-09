package postgres

import (
	"context"
	"sort"

	"gorm.io/gorm"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ProductRepository implements port.ProductRepository.
type ProductRepository struct{ db *DB }

func NewProductRepository(db *DB) *ProductRepository { return &ProductRepository{db: db} }

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	m := fromProduct(p)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	p.CreatedAt, p.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	// Every mutable column must be listed: an omitted one is silently dropped on
	// update (there is a regression test for this).
	res := r.db.session(ctx).Model(&productModel{}).Where("id = ?", p.ID).Updates(map[string]any{
		"category_id": nullableUUID(p.CategoryID),
		"title":       p.Title,
		// jsonMap keeps a nil map from writing SQL NULL into a NOT NULL column.
		"names":        jsonMap(p.Names),
		"slug":         p.Slug,
		"description":  p.Description,
		"price_cents":  p.PriceCents,
		"cost_cents":   p.CostCents,
		"weight_grams": p.WeightGrams,
		"currency":     p.Currency,
		"cover_image":  p.CoverImage,
		"images":       stringList(p.Images),
		"status":       string(p.Status),
		"stock":        p.Stock,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	var m productModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toProduct(&m), nil
}

func (r *ProductRepository) FindBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var m productModel
	if err := r.db.session(ctx).First(&m, "slug = ?", slug).Error; err != nil {
		return nil, translate(err)
	}
	return toProduct(&m), nil
}

func (r *ProductRepository) List(ctx context.Context, f domain.ProductFilter) (domain.Page[domain.Product], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)

	q := r.db.session(ctx).Model(&productModel{})
	if f.CategoryID != "" {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.Keyword != "" {
		if f.Fuzzy {
			where, args := fuzzyClause(f.Keyword)
			q = q.Where(where, args...)
		} else {
			where, args := searchClause(f.Keyword, expandSynonyms(f.Keyword, f.Synonyms))
			q = q.Where(where, args...)
		}
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if f.MinPriceCents != nil {
		q = q.Where("price_cents >= ?", *f.MinPriceCents)
	}
	if f.MaxPriceCents != nil {
		q = q.Where("price_cents <= ?", *f.MaxPriceCents)
	}
	if len(f.Attributes) > 0 {
		// Match a variant carrying every requested attribute. Variants are
		// scanned per product, so this is an EXISTS rather than a join (which
		// would multiply the product rows).
		q = q.Where(
			"EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = products.id AND v.attributes @> ?)",
			jsonMap(f.Attributes),
		)
	}

	// A keyword search defaults to relevance so the best matches come first.
	relevance := f.Keyword != "" && (f.Sort == "" || f.Sort == "relevance")

	// Keyset (cursor) mode: stable and O(1) at any depth. Only the newest
	// ordering is supported; other sorts fall back to offset paging.
	// Relevance ranking cannot be keyset paginated: the rank depends on the
	// keyword, so offset paging is used instead.
	if f.CursorMode && !relevance && (f.Sort == "" || f.Sort == "newest") {
		if f.Cursor != "" {
			createdAt, id, err := decodeCursor(f.Cursor)
			if err != nil {
				return domain.Page[domain.Product]{}, err
			}
			q = q.Where("(created_at, id) < (?, ?)", createdAt, id)
		}
		var models []productModel
		err := q.Order("created_at desc, id desc").Limit(size + 1).Find(&models).Error
		if err != nil {
			return domain.Page[domain.Product]{}, translate(err)
		}
		var next string
		if len(models) > size {
			models = models[:size]
			last := models[len(models)-1]
			next = encodeCursor(last.CreatedAt, last.ID)
		}
		items := make([]domain.Product, 0, len(models))
		for i := range models {
			items = append(items, *toProduct(&models[i]))
		}
		return domain.Page[domain.Product]{Items: items, PageSize: size, NextCursor: next}, nil
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Product]{}, translate(err)
	}

	switch {
	case f.Fuzzy:
		q = q.Order(fuzzyOrder(f.Keyword)).Order("created_at desc, id desc")
	case relevance:
		q = q.Order(relevanceOrder(f.Keyword)).Order("created_at desc, id desc")
	case f.Sort == "price_asc":
		q = q.Order("price_cents asc, id desc")
	case f.Sort == "price_desc":
		q = q.Order("price_cents desc, id desc")
	default:
		q = q.Order("created_at desc, id desc")
	}

	var models []productModel
	if err := q.Offset((page - 1) * size).Limit(size).Find(&models).Error; err != nil {
		return domain.Page[domain.Product]{}, translate(err)
	}
	items := make([]domain.Product, 0, len(models))
	for i := range models {
		items = append(items, *toProduct(&models[i]))
	}
	return domain.Page[domain.Product]{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// DecreaseStock decrements stock atomically. The WHERE guard makes the update a
// compare-and-set, so concurrent checkouts across instances cannot oversell.
func (r *ProductRepository) DecreaseStock(ctx context.Context, productID string, quantity int) error {
	res := r.db.session(ctx).Model(&productModel{}).
		Where("id = ? AND stock >= ?", productID, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrInsufficientStock
	}
	return nil
}

func (r *ProductRepository) IncreaseStock(ctx context.Context, productID string, quantity int) error {
	res := r.db.session(ctx).Model(&productModel{}).
		Where("id = ?", productID).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func normalizePage(page, size, defSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defSize
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

var _ port.ProductRepository = (*ProductRepository)(nil)

// Facets returns the price range and the variant attributes that occur in a
// category. Values are collected in Go because the attribute map is free-form
// JSON: an operator can invent any attribute without a migration.
func (r *ProductRepository) Facets(ctx context.Context, categoryID string) (domain.ProductFacets, error) {
	out := domain.ProductFacets{Attributes: []domain.AttributeFacet{}}

	bounds := r.db.session(ctx).Model(&productModel{}).Where("status = ?", string(domain.ProductPublished))
	if categoryID != "" {
		bounds = bounds.Where("category_id = ?", categoryID)
	}
	var rangeRow struct {
		Min int64
		Max int64
	}
	if err := bounds.Select("coalesce(min(price_cents), 0) AS min, coalesce(max(price_cents), 0) AS max").
		Scan(&rangeRow).Error; err != nil {
		return out, translate(err)
	}
	out.MinPriceCents, out.MaxPriceCents = rangeRow.Min, rangeRow.Max

	var rows []struct {
		Attributes jsonMap
	}
	q := r.db.session(ctx).Table("product_variants AS v").
		Joins("JOIN products p ON p.id = v.product_id").
		Where("p.status = ? AND v.active = true", string(domain.ProductPublished)).
		Where("jsonb_typeof(v.attributes) = 'object'")
	if categoryID != "" {
		q = q.Where("p.category_id = ?", categoryID)
	}
	if err := q.Select("v.attributes").Scan(&rows).Error; err != nil {
		return out, translate(err)
	}

	values := map[string]map[string]bool{}
	for _, row := range rows {
		for name, value := range row.Attributes {
			if name == "" || value == "" {
				continue
			}
			if values[name] == nil {
				values[name] = map[string]bool{}
			}
			values[name][value] = true
		}
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		facet := domain.AttributeFacet{Name: name, Values: make([]string, 0, len(values[name]))}
		for value := range values[name] {
			facet.Values = append(facet.Values, value)
		}
		sort.Strings(facet.Values)
		out.Attributes = append(out.Attributes, facet)
	}
	return out, nil
}
