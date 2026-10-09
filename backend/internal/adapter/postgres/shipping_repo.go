package postgres

import (
	"context"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ShippingMethodRepository implements port.ShippingMethodRepository.
type ShippingMethodRepository struct{ db *DB }

func NewShippingMethodRepository(db *DB) *ShippingMethodRepository {
	return &ShippingMethodRepository{db: db}
}

func (r *ShippingMethodRepository) Create(ctx context.Context, m *domain.ShippingMethod) error {
	model := fromShippingMethod(m)
	if err := r.db.session(ctx).Create(model).Error; err != nil {
		return translate(err)
	}
	m.CreatedAt, m.UpdatedAt = model.CreatedAt, model.UpdatedAt
	return nil
}

func (r *ShippingMethodRepository) Update(ctx context.Context, m *domain.ShippingMethod) error {
	res := r.db.session(ctx).Model(&shippingMethodModel{}).Where("id = ?", m.ID).Updates(map[string]any{
		"code":                 m.Code,
		"name":                 m.Name,
		"flat_rate_cents":      m.FlatRateCents,
		"min_days":             m.MinDays,
		"max_days":             m.MaxDays,
		"free_threshold_cents": m.FreeThresholdCents,
		"active":               m.Active,
		"sort":                 m.Sort,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ShippingMethodRepository) FindByID(ctx context.Context, id string) (*domain.ShippingMethod, error) {
	var m shippingMethodModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toShippingMethod(&m), nil
}

func (r *ShippingMethodRepository) List(ctx context.Context, activeOnly bool) ([]domain.ShippingMethod, error) {
	q := r.db.session(ctx).Model(&shippingMethodModel{})
	if activeOnly {
		q = q.Where("active")
	}
	var models []shippingMethodModel
	if err := q.Order("sort asc, created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.ShippingMethod, 0, len(models))
	for i := range models {
		out = append(out, *toShippingMethod(&models[i]))
	}
	return out, nil
}

func (r *ShippingMethodRepository) Default(ctx context.Context) (*domain.ShippingMethod, error) {
	var m shippingMethodModel
	if err := r.db.session(ctx).Where("active").Order("sort asc, created_at asc").First(&m).Error; err != nil {
		return nil, translate(err)
	}
	return toShippingMethod(&m), nil
}

var _ port.ShippingMethodRepository = (*ShippingMethodRepository)(nil)
