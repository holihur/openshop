package postgres

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/holihur/openshop/internal/domain"
	"github.com/holihur/openshop/internal/port"
)

// ShippingZoneRepository implements port.ShippingZoneRepository.
type ShippingZoneRepository struct{ db *DB }

func NewShippingZoneRepository(db *DB) *ShippingZoneRepository {
	return &ShippingZoneRepository{db: db}
}

func (r *ShippingZoneRepository) Create(ctx context.Context, z *domain.ShippingZone) error {
	m := fromShippingZone(z)
	if err := r.db.session(ctx).Create(m).Error; err != nil {
		return translate(err)
	}
	z.CreatedAt, z.UpdatedAt = m.CreatedAt, m.UpdatedAt
	return nil
}

func (r *ShippingZoneRepository) Update(ctx context.Context, z *domain.ShippingZone) error {
	res := r.db.session(ctx).Model(&shippingZoneModel{}).Where("id = ?", z.ID).Updates(map[string]any{
		"name":      z.Name,
		"provinces": stringList(z.Provinces),
		"active":    z.Active,
		"sort":      z.Sort,
	})
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ShippingZoneRepository) List(ctx context.Context, activeOnly bool) ([]domain.ShippingZone, error) {
	q := r.db.session(ctx).Model(&shippingZoneModel{})
	if activeOnly {
		q = q.Where("active")
	}
	var models []shippingZoneModel
	if err := q.Order("sort asc, created_at asc").Find(&models).Error; err != nil {
		return nil, translate(err)
	}
	out := make([]domain.ShippingZone, 0, len(models))
	for i := range models {
		out = append(out, *toShippingZone(&models[i]))
	}
	return out, nil
}

func (r *ShippingZoneRepository) FindByID(ctx context.Context, id string) (*domain.ShippingZone, error) {
	var m shippingZoneModel
	if err := r.db.session(ctx).First(&m, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}
	return toShippingZone(&m), nil
}

// FindByProvince loads active zones in priority order and returns the first that
// covers the province. Province lists are small, so matching in Go is fine.
func (r *ShippingZoneRepository) FindByProvince(ctx context.Context, province string) (*domain.ShippingZone, error) {
	zones, err := r.List(ctx, true)
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].Matches(province) {
			return &zones[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ShippingZoneRepository) UpsertRate(ctx context.Context, rate *domain.ShippingRate) error {
	m := fromShippingRate(rate)
	err := r.db.session(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "zone_id"}, {Name: "method_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"flat_rate_cents", "free_threshold_cents", "per_kg_cents", "min_days", "max_days"}),
	}).Create(m).Error
	return translate(err)
}

func (r *ShippingZoneRepository) FindRate(ctx context.Context, zoneID, methodID string) (*domain.ShippingRate, error) {
	var m shippingRateModel
	if err := r.db.session(ctx).First(&m, "zone_id = ? AND method_id = ?", zoneID, methodID).Error; err != nil {
		return nil, translate(err)
	}
	return toShippingRate(&m), nil
}

var _ port.ShippingZoneRepository = (*ShippingZoneRepository)(nil)
