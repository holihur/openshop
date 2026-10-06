package postgres

import (
	"context"

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
	res := r.db.session(ctx).Model(&productModel{}).Where("id = ?", p.ID).Updates(map[string]any{
		"category_id": nullableUUID(p.CategoryID),
		"title":       p.Title,
		"slug":        p.Slug,
		"description": p.Description,
		"price_cents": p.PriceCents,
		"currency":    p.Currency,
		"cover_image": p.CoverImage,
		"images":      stringList(p.Images),
		"status":      string(p.Status),
		"stock":       p.Stock,
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

func (r *ProductRepository) List(ctx context.Context, f domain.ProductFilter) (domain.Page[domain.Product], error) {
	page, size := normalizePage(f.Page, f.PageSize, 20)

	q := r.db.session(ctx).Model(&productModel{})
	if f.CategoryID != "" {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		q = q.Where("title ILIKE ? OR description ILIKE ?", like, like)
	}
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return domain.Page[domain.Product]{}, translate(err)
	}

	switch f.Sort {
	case "price_asc":
		q = q.Order("price_cents asc")
	case "price_desc":
		q = q.Order("price_cents desc")
	default:
		q = q.Order("created_at desc")
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
