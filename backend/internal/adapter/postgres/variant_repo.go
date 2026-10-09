package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// VariantRepository implements port.VariantRepository.
type VariantRepository struct{ db *DB }

func NewVariantRepository(db *DB) *VariantRepository { return &VariantRepository{db: db} }

func (r *VariantRepository) Create(ctx context.Context, v *domain.Variant) error {
	m := fromVariant(v)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	v.CreatedAt, v.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *VariantRepository) Update(ctx context.Context, v *domain.Variant) error {
	// Every mutable column must be listed (there is a regression test).
	res := r.db.session(ctx).Model(&variantModel{}).Where("id = ?", v.ID).Updates(map[string]any{
		"sku":          v.SKU,
		"name":         v.Name,
		"price_cents":  v.PriceCents,
		"cost_cents":   v.CostCents,
		"stock":        v.Stock,
		"weight_grams": v.WeightGrams,
		"attributes":   jsonMap(v.Attributes),
		"sort":         v.Sort,
		"active":       v.Active,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *VariantRepository) FindByID(ctx context.Context, id string) (*domain.Variant, error) {
	var m variantModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toVariant(&m), nil
}

func (r *VariantRepository) ListByProduct(ctx context.Context, productID string) ([]domain.Variant, error) {
	var models []variantModel
	if err := r.db.session(ctx).Where("product_id = ?", productID).
		Order("sort asc, created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.Variant, 0, len(models))
	for i := range models {
		out = append(out, *toVariant(&models[i]))
	}
	return out, nil
}

// DecreaseStock is a compare-and-set that prevents overselling a variant across
// concurrent checkouts on any replica.
func (r *VariantRepository) DecreaseStock(ctx context.Context, variantID string, quantity int) error {
	res := r.db.session(ctx).Model(&variantModel{}).
		Where("id = ? AND stock >= ?", variantID, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrInsufficientStock
	}
	return nil
}

func (r *VariantRepository) IncreaseStock(ctx context.Context, variantID string, quantity int) error {
	res := r.db.session(ctx).Model(&variantModel{}).
		Where("id = ?", variantID).
		UpdateColumn("stock", gorm.Expr("stock + ?", quantity))
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ port.VariantRepository = (*VariantRepository)(nil)
